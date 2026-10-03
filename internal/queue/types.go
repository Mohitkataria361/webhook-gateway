package queue

import "encoding/json"

const (
	// ExchangeName is the RabbitMQ topic exchange used for routing webhook jobs.
	ExchangeName = "webhook.events"

	// QueueName is the main delivery queue.
	QueueName = "webhook.delivery"

	// DLQName is the Dead-Letter Queue for exhausted-retry events.
	DLQName = "webhook.dlq"

	// RoutingKey used when publishing jobs to the main exchange.
	RoutingKey = "webhook.dispatch"
)

// WebhookJob is the message payload published to RabbitMQ.
// Workers deserialize this struct from AMQP message bodies.
type WebhookJob struct {
	EventID        string          `json:"event_id"`
	EndpointID     string          `json:"endpoint_id"`
	TargetURL      string          `json:"target_url"`
	SecretKey      string          `json:"secret_key"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	AttemptNumber  int             `json:"attempt_number"`
	MaxRetries     int             `json:"max_retries"`
}

// Encode serializes the job to JSON bytes for AMQP publishing.
func (j *WebhookJob) Encode() ([]byte, error) {
	return json.Marshal(j)
}

// DecodeJob deserializes AMQP message body into a WebhookJob.
func DecodeJob(body []byte) (*WebhookJob, error) {
	var job WebhookJob
	if err := json.Unmarshal(body, &job); err != nil {
		return nil, err
	}
	return &job, nil
}
