package worker

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"webhook-gateway/internal/hub"
	"webhook-gateway/internal/models"
	"webhook-gateway/internal/queue"
	redisclient "webhook-gateway/internal/redis"
	"webhook-gateway/internal/signer"

	"github.com/jackc/pgx/v5/pgxpool"
)

const httpTimeout = 10 * time.Second

// Dispatcher sends a webhook HTTP request for a given job.
type Dispatcher struct {
	db          *pgxpool.Pool
	redisClient *redisclient.Client
	hub         *hub.Hub
	httpClient  *http.Client
}

// NewDispatcher creates a Dispatcher with a shared http.Client.
func NewDispatcher(db *pgxpool.Pool, rc *redisclient.Client, h *hub.Hub) *Dispatcher {
	return &Dispatcher{
		db:          db,
		redisClient: rc,
		hub:         h,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// Dispatch executes a single webhook delivery attempt.
// It checks the circuit breaker, signs the payload, dispatches the HTTP request,
// records the result, and broadcasts to the WebSocket hub.
func (d *Dispatcher) Dispatch(ctx context.Context, job *queue.WebhookJob, mq *queue.RabbitMQ) error {
	// --- Step 1: Circuit Breaker Check ---
	isOpen, err := d.redisClient.IsCircuitOpen(ctx, job.EndpointID)
	if err != nil {
		log.Printf("[dispatcher] Circuit breaker check failed for %s: %v", job.EndpointID, err)
	}
	if isOpen {
		log.Printf("[dispatcher] Circuit OPEN for endpoint %s — skipping dispatch", job.EndpointID)
		errMsg := "circuit breaker open"
		d.saveDelivery(ctx, job, nil, nil, nil, models.DeliveryCircuitBroken, &errMsg)
		d.broadcastStatus(job, nil, nil, models.DeliveryCircuitBroken, &errMsg)
		return nil
	}

	// --- Step 2: Sign the payload ---
	payloadBytes := []byte(job.Payload)
	signature := signer.Sign(job.SecretKey, payloadBytes)

	// --- Step 3: Build and send the HTTP request ---
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.TargetURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", signature)
	req.Header.Set("X-Webhook-Event-ID", job.EventID)
	req.Header.Set("X-Webhook-Attempt", fmt.Sprintf("%d", job.AttemptNumber))

	startTime := time.Now()
	resp, err := d.httpClient.Do(req)
	elapsedMS := int(time.Since(startTime).Milliseconds())

	// --- Step 4: Handle network error (timeout, DNS, etc.) ---
	if err != nil {
		errMsg := err.Error()
		log.Printf("[dispatcher] HTTP error for event %s (attempt %d): %v", job.EventID, job.AttemptNumber, err)
		_ = d.redisClient.RecordFailure(ctx, job.EndpointID)
		return d.handleRetryOrDLQ(ctx, job, mq, nil, &elapsedMS, &errMsg)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyStr := string(bodyBytes)
	status := resp.StatusCode

	// --- Step 5: Evaluate response status ---
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// SUCCESS
		log.Printf("[dispatcher] SUCCESS event %s (attempt %d) → %d in %dms", job.EventID, job.AttemptNumber, status, elapsedMS)
		_ = d.redisClient.RecordSuccess(ctx, job.EndpointID)
		d.saveDelivery(ctx, job, &status, &bodyStr, &elapsedMS, models.DeliverySuccess, nil)
		d.broadcastStatus(job, &status, &elapsedMS, models.DeliverySuccess, nil)
		d.updateEventStatus(ctx, job.EventID, "DELIVERED")
		return nil
	}

	// 5xx or 4xx — treat as failure and retry.
	errMsg := fmt.Sprintf("non-2xx response: %d", status)
	log.Printf("[dispatcher] FAILED event %s (attempt %d) → %d: %s", job.EventID, job.AttemptNumber, status, bodyStr)
	_ = d.redisClient.RecordFailure(ctx, job.EndpointID)
	return d.handleRetryOrDLQ(ctx, job, mq, &status, &elapsedMS, &errMsg)
}

// handleRetryOrDLQ re-queues with backoff or routes to DLQ on exhaustion.
func (d *Dispatcher) handleRetryOrDLQ(
	ctx context.Context,
	job *queue.WebhookJob,
	mq *queue.RabbitMQ,
	httpStatus *int,
	elapsedMS *int,
	errMsg *string,
) error {
	if job.AttemptNumber >= job.MaxRetries {
		log.Printf("[dispatcher] Max retries reached for event %s — sending to DLQ", job.EventID)
		bodyStr := "max retries exhausted"
		d.saveDelivery(ctx, job, httpStatus, &bodyStr, elapsedMS, models.DeliveryDLQ, errMsg)
		d.broadcastStatus(job, httpStatus, elapsedMS, models.DeliveryDLQ, errMsg)
		d.updateEventStatus(ctx, job.EventID, "DLQ")

		dlqJob := *job
		dlqJob.AttemptNumber++
		return mq.PublishToDLQ(ctx, &dlqJob)
	}

	// Re-queue with incremented attempt count (caller handles backoff sleep).
	retryJob := *job
	retryJob.AttemptNumber++
	d.saveDelivery(ctx, job, httpStatus, nil, elapsedMS, models.DeliveryFailed, errMsg)
	d.broadcastStatus(job, httpStatus, elapsedMS, models.DeliveryFailed, errMsg)

	// Apply exponential backoff before re-publishing.
	delay := Backoff(retryJob.AttemptNumber)
	log.Printf("[dispatcher] Retrying event %s in %v (attempt %d)", job.EventID, delay, retryJob.AttemptNumber)
	time.Sleep(delay)

	return mq.Publish(ctx, &retryJob)
}

// saveDelivery inserts a webhook_deliveries record into PostgreSQL.
func (d *Dispatcher) saveDelivery(
	ctx context.Context,
	job *queue.WebhookJob,
	httpStatus *int,
	responseBody *string,
	execTimeMS *int,
	status string,
	errMsg *string,
) {
	_, err := d.db.Exec(ctx, `
		INSERT INTO webhook_deliveries
			(event_id, attempt_number, http_status, response_body, execution_time_ms, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		job.EventID, job.AttemptNumber, httpStatus, responseBody, execTimeMS, status, errMsg,
	)
	if err != nil {
		log.Printf("[dispatcher] Failed to save delivery record: %v", err)
	}
}

// updateEventStatus sets the top-level status on the webhook_events record.
func (d *Dispatcher) updateEventStatus(ctx context.Context, eventID, status string) {
	_, err := d.db.Exec(ctx,
		"UPDATE webhook_events SET status = $1 WHERE id = $2",
		status, eventID,
	)
	if err != nil {
		log.Printf("[dispatcher] Failed to update event status: %v", err)
	}
}

// broadcastStatus publishes a live update to the WebSocket hub.
func (d *Dispatcher) broadcastStatus(
	job *queue.WebhookJob,
	httpStatus *int,
	execTimeMS *int,
	status string,
	errMsg *string,
) {
	d.hub.Broadcast(hub.LogEvent{
		EventID:         job.EventID,
		EndpointID:      job.EndpointID,
		TargetURL:       job.TargetURL,
		AttemptNumber:   job.AttemptNumber,
		HTTPStatus:      httpStatus,
		Status:          status,
		ExecutionTimeMS: execTimeMS,
		ErrorMessage:    errMsg,
	})
}
