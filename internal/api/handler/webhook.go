package handler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"webhook-gateway/internal/config"
	"webhook-gateway/internal/queue"
	redisclient "webhook-gateway/internal/redis"
)

// WebhookHandler handles event ingestion, deliveries, logs, and replay.
type WebhookHandler struct {
	db    *pgxpool.Pool
	redis *redisclient.Client
	mq    *queue.RabbitMQ
	cfg   *config.Config
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(db *pgxpool.Pool, rc *redisclient.Client, mq *queue.RabbitMQ, cfg *config.Config) *WebhookHandler {
	return &WebhookHandler{db: db, redis: rc, mq: mq, cfg: cfg}
}

// SendRequest is the JSON body for POST /api/v1/webhooks/send.
type SendRequest struct {
	EndpointID     string          `json:"endpoint_id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key"`
}

// Send handles POST /api/v1/webhooks/send.
func (h *WebhookHandler) Send(c *fiber.Ctx) error {
	var req SendRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}

	// Basic validation.
	if req.EndpointID == "" || req.EventType == "" || req.IdempotencyKey == "" || len(req.Payload) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "endpoint_id, event_type, payload, and idempotency_key are required",
		})
	}

	endpointID, err := uuid.Parse(req.EndpointID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "endpoint_id must be a valid UUID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// --- Idempotency Check (Redis SETNX) ---
	isNew, err := h.redis.SetIdempotency(ctx, req.IdempotencyKey)
	if err != nil {
		log.Printf("[webhook-handler] Redis idempotency error: %v", err)
	}
	if !isNew {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":           "duplicate request",
			"idempotency_key": req.IdempotencyKey,
		})
	}

	// --- Load endpoint config and check user ownership ---
	var (
		targetURL, secretKey string
		endpointUserID       uuid.UUID
	)

	authUID, _ := c.Locals("user_id").(string)
	if authUID != "" {
		callerUID, err := uuid.Parse(authUID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id in token"})
		}

		err = h.db.QueryRow(ctx,
			"SELECT target_url, secret_key, user_id FROM webhook_endpoints WHERE id = $1 AND user_id = $2 AND is_active = TRUE",
			endpointID, callerUID,
		).Scan(&targetURL, &secretKey, &endpointUserID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "endpoint not found or not owned by you"})
		}
	} else {
		err = h.db.QueryRow(ctx,
			"SELECT target_url, secret_key, user_id FROM webhook_endpoints WHERE id = $1 AND is_active = TRUE",
			endpointID,
		).Scan(&targetURL, &secretKey, &endpointUserID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "endpoint not found or inactive"})
		}
	}

	// --- Persist the event ---
	var eventID string
	err = h.db.QueryRow(ctx, `
		INSERT INTO webhook_events (endpoint_id, event_type, payload, idempotency_key, status)
		VALUES ($1, $2, $3, $4, 'PENDING')
		RETURNING id`,
		endpointID, req.EventType, req.Payload, req.IdempotencyKey,
	).Scan(&eventID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to persist event: " + err.Error()})
	}

	// --- Publish to RabbitMQ with UserID ---
	job := &queue.WebhookJob{
		EventID:       eventID,
		EndpointID:    endpointID.String(),
		UserID:        endpointUserID.String(),
		TargetURL:     targetURL,
		SecretKey:     secretKey,
		EventType:     req.EventType,
		Payload:       req.Payload,
		AttemptNumber: 1,
		MaxRetries:    h.cfg.MaxRetries,
	}

	if err := h.mq.Publish(ctx, job); err != nil {
		log.Printf("[webhook-handler] Failed to publish job for event %s: %v", eventID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to queue event"})
	}

	log.Printf("[webhook-handler] Enqueued event %s (user=%s, type=%s, endpoint=%s)", eventID, endpointUserID, req.EventType, req.EndpointID)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"event_id": eventID,
		"status":   "PENDING",
		"message":  "Webhook event accepted and queued for delivery",
	})
}

// GetDeliveries handles GET /api/v1/events/:id/deliveries.
// Returns delivery attempts for a given event ID owned by the user.
func (h *WebhookHandler) GetDeliveries(c *fiber.Ctx) error {
	eventIDStr := c.Params("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authUID, _ := c.Locals("user_id").(string)
	query := `
		SELECT d.id, d.event_id, d.attempt_number, d.http_status, d.response_body,
		       d.execution_time_ms, d.status, d.error_message, d.delivered_at
		FROM webhook_deliveries d
		JOIN webhook_events e ON d.event_id = e.id
		JOIN webhook_endpoints ep ON e.endpoint_id = ep.id
		WHERE event_id = $1`

	args := []interface{}{eventID}
	if authUID != "" {
		callerUID, err := uuid.Parse(authUID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id in token"})
		}
		query += " AND ep.user_id = $2"
		args = append(args, callerUID)
	}
	query += " ORDER BY attempt_number ASC"

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	type DeliveryRow struct {
		ID              string    `json:"id"`
		EventID         string    `json:"event_id"`
		AttemptNumber   int       `json:"attempt_number"`
		HTTPStatus      *int      `json:"http_status"`
		ResponseBody    *string   `json:"response_body"`
		ExecutionTimeMS *int      `json:"execution_time_ms"`
		Status          string    `json:"status"`
		ErrorMessage    *string   `json:"error_message"`
		DeliveredAt     time.Time `json:"delivered_at"`
	}

	var deliveries []DeliveryRow
	for rows.Next() {
		var d DeliveryRow
		if err := rows.Scan(
			&d.ID, &d.EventID, &d.AttemptNumber, &d.HTTPStatus, &d.ResponseBody,
			&d.ExecutionTimeMS, &d.Status, &d.ErrorMessage, &d.DeliveredAt,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		deliveries = append(deliveries, d)
	}

	return c.JSON(fiber.Map{"event_id": eventID, "deliveries": deliveries, "count": len(deliveries)})
}

// GetLogs handles GET /api/v1/logs.
// Returns historical delivery logs strictly scoped to the authenticated user's endpoints.
func (h *WebhookHandler) GetLogs(c *fiber.Ctx) error {
	authUID, _ := c.Locals("user_id").(string)
	if authUID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	callerUID, err := uuid.Parse(authUID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT d.id, d.event_id, e.endpoint_id, ep.target_url, d.attempt_number,
		       d.http_status, d.response_body, d.execution_time_ms, d.status,
		       d.error_message, d.delivered_at
		FROM webhook_deliveries d
		JOIN webhook_events e ON d.event_id = e.id
		JOIN webhook_endpoints ep ON e.endpoint_id = ep.id
		WHERE ep.user_id = $1
		ORDER BY d.delivered_at DESC
		LIMIT 100`,
		callerUID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	type LogItem struct {
		ID              string    `json:"id"`
		EventID         string    `json:"event_id"`
		EndpointID      string    `json:"endpoint_id"`
		TargetURL       string    `json:"target_url"`
		AttemptNumber   int       `json:"attempt_number"`
		HTTPStatus      *int      `json:"http_status"`
		ResponseBody    *string   `json:"response_body"`
		ExecutionTimeMS *int      `json:"execution_time_ms"`
		Status          string    `json:"status"`
		ErrorMessage    *string   `json:"error_message"`
		DeliveredAt     time.Time `json:"delivered_at"`
	}

	var logsList []LogItem
	for rows.Next() {
		var l LogItem
		if err := rows.Scan(
			&l.ID, &l.EventID, &l.EndpointID, &l.TargetURL, &l.AttemptNumber,
			&l.HTTPStatus, &l.ResponseBody, &l.ExecutionTimeMS, &l.Status,
			&l.ErrorMessage, &l.DeliveredAt,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		logsList = append(logsList, l)
	}

	return c.JSON(fiber.Map{"logs": logsList, "count": len(logsList)})
}

// Replay handles POST /api/v1/events/:id/replay.
// Re-queues an existing event owned by the authenticated user.
func (h *WebhookHandler) Replay(c *fiber.Ctx) error {
	eventIDStr := c.Params("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authUID, _ := c.Locals("user_id").(string)
	query := `
		SELECT e.id, e.endpoint_id, ep.user_id, e.event_type, e.payload, ep.target_url, ep.secret_key
		FROM webhook_events e
		JOIN webhook_endpoints ep ON ep.id = e.endpoint_id
		WHERE e.id = $1`

	args := []interface{}{eventID}
	if authUID != "" {
		callerUID, err := uuid.Parse(authUID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id in token"})
		}
		query += " AND ep.user_id = $2"
		args = append(args, callerUID)
	}

	var (
		evtIDStr, endpointIDStr, eventType, targetURL, secretKey string
		endpointUserID                                           uuid.UUID
		payload                                                  json.RawMessage
	)
	err = h.db.QueryRow(ctx, query, args...).Scan(
		&evtIDStr, &endpointIDStr, &endpointUserID, &eventType, &payload, &targetURL, &secretKey,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found or not owned by you"})
	}

	// Reset event status to PENDING.
	h.db.Exec(ctx, "UPDATE webhook_events SET status = 'PENDING' WHERE id = $1", eventID)

	job := &queue.WebhookJob{
		EventID:       evtIDStr,
		EndpointID:    endpointIDStr,
		UserID:        endpointUserID.String(),
		TargetURL:     targetURL,
		SecretKey:     secretKey,
		EventType:     eventType,
		Payload:       payload,
		AttemptNumber: 1,
		MaxRetries:    h.cfg.MaxRetries,
	}

	if err := h.mq.Publish(ctx, job); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to re-queue event"})
	}

	log.Printf("[webhook-handler] Replayed event %s for user %s", evtIDStr, endpointUserID)
	return c.JSON(fiber.Map{"event_id": evtIDStr, "status": "PENDING", "message": "Event re-queued for replay"})
}
