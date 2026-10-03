package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebhookEvent is the master record for an ingested webhook payload.
type WebhookEvent struct {
	ID             uuid.UUID       `json:"id"`
	EndpointID     uuid.UUID       `json:"endpoint_id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key"`
	Status         string          `json:"status"` // PENDING | DELIVERED | FAILED | DLQ
	CreatedAt      time.Time       `json:"created_at"`
}
