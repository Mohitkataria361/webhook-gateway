package models

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryStatus enumerates the possible states of a delivery attempt.
const (
	DeliverySuccess       = "SUCCESS"
	DeliveryFailed        = "FAILED"
	DeliveryCircuitBroken = "CIRCUIT_BROKEN"
	DeliveryDLQ           = "DLQ"
)

// WebhookDelivery records a single delivery attempt for a WebhookEvent.
type WebhookDelivery struct {
	ID              uuid.UUID  `json:"id"`
	EventID         uuid.UUID  `json:"event_id"`
	AttemptNumber   int        `json:"attempt_number"`
	HTTPStatus      *int       `json:"http_status,omitempty"`
	ResponseBody    *string    `json:"response_body,omitempty"`
	ExecutionTimeMS *int       `json:"execution_time_ms,omitempty"`
	Status          string     `json:"status"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	DeliveredAt     time.Time  `json:"delivered_at"`
}
