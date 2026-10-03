package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EndpointHandler handles endpoint registration and management.
type EndpointHandler struct {
	db *pgxpool.Pool
}

// NewEndpointHandler creates an EndpointHandler.
func NewEndpointHandler(db *pgxpool.Pool) *EndpointHandler {
	return &EndpointHandler{db: db}
}

// RegisterRequest is the JSON body for POST /api/v1/endpoints.
type RegisterRequest struct {
	UserID    string `json:"user_id"    validate:"required"`
	TargetURL string `json:"target_url" validate:"required,url"`
}

// RegisterResponse includes the generated secret key — returned ONCE at registration.
type RegisterResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TargetURL string    `json:"target_url"`
	SecretKey string    `json:"secret_key"` // Store this securely — not returned again
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Register handles POST /api/v1/endpoints.
// Generates a cryptographically random 32-byte secret key for HMAC signing.
func (h *EndpointHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}
	if req.UserID == "" || req.TargetURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id and target_url are required"})
	}

	// Validate user_id is a valid UUID.
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id must be a valid UUID"})
	}

	// Generate a 32-byte cryptographic secret key.
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate secret"})
	}
	secretKey := hex.EncodeToString(secretBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resp RegisterResponse
	err = h.db.QueryRow(ctx, `
		INSERT INTO webhook_endpoints (user_id, target_url, secret_key)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, target_url, secret_key, is_active, created_at`,
		userID, req.TargetURL, secretKey,
	).Scan(&resp.ID, &resp.UserID, &resp.TargetURL, &resp.SecretKey, &resp.IsActive, &resp.CreatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// List handles GET /api/v1/endpoints?user_id=<uuid>.
func (h *EndpointHandler) List(c *fiber.Ctx) error {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id query param required"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id must be a valid UUID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx,
		"SELECT id, user_id, target_url, is_active, created_at FROM webhook_endpoints WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	type EndpointItem struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
		TargetURL string    `json:"target_url"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
	}

	var endpoints []EndpointItem
	for rows.Next() {
		var e EndpointItem
		if err := rows.Scan(&e.ID, &e.UserID, &e.TargetURL, &e.IsActive, &e.CreatedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		endpoints = append(endpoints, e)
	}

	return c.JSON(fiber.Map{"endpoints": endpoints, "count": len(endpoints)})
}
