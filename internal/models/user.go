package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated gateway user.
type User struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	AvatarURL      string    `json:"avatar_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
