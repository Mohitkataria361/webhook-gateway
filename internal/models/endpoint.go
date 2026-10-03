package models

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEndpoint represents a registered customer target URL with its signing secret.
type WebhookEndpoint struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TargetURL string    `json:"target_url"`
	SecretKey string    `json:"-"`          // Never expose in API responses
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
