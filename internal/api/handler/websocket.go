package handler

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"webhook-gateway/internal/auth"
	"webhook-gateway/internal/config"
	"webhook-gateway/internal/hub"
)

// WebSocketHandler manages WebSocket upgrade and authenticated client lifecycle.
type WebSocketHandler struct {
	db  *pgxpool.Pool
	hub *hub.Hub
	cfg *config.Config
}

// NewWebSocketHandler creates a WebSocketHandler.
func NewWebSocketHandler(db *pgxpool.Pool, h *hub.Hub, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{db: db, hub: h, cfg: cfg}
}

// UpgradeMiddleware validates that the request is a WebSocket upgrade request
// and verifies the API key or JWT token from query param ?token=... / ?api_key=..., Authorization header, or cookie.
func (h *WebSocketHandler) UpgradeMiddleware(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	token := c.Query("token")
	if token == "" {
		token = c.Query("api_key")
	}
	if token == "" {
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		}
	}
	if token == "" {
		token = c.Cookies("auth_token")
	}

	if token == "" {
		log.Println("[ws] Upgrade rejected: missing auth token or API key")
		return fiber.ErrUnauthorized
	}

	// 1. Try API key lookup if DB is configured
	if h.db != nil {
		keyHash := auth.HashAPIKey(token)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var userID uuid.UUID
		err := h.db.QueryRow(ctx,
			"SELECT user_id FROM api_keys WHERE key_hash = $1 AND is_active = TRUE",
			keyHash,
		).Scan(&userID)

		if err == nil {
			c.Locals("user_id", userID.String())
			return c.Next()
		}
	}

	// 2. Try JWT validation
	if h.cfg.JWTSecret != "" {
		claims, err := auth.ValidateToken(token, h.cfg.JWTSecret)
		if err == nil {
			c.Locals("user_id", claims.UserID)
			c.Locals("email", claims.Email)
			return c.Next()
		}
	}

	log.Printf("[ws] Upgrade rejected: invalid credentials")
	return fiber.ErrUnauthorized
}

// Handle is the Fiber WebSocket handler for /ws/logs.
// Registers the client with the hub scoped to the authenticated user ID.
func (h *WebSocketHandler) Handle(c *websocket.Conn) {
	userID, _ := c.Locals("user_id").(string)
	h.hub.Register(c, userID)
	defer h.hub.Unregister(c)

	log.Printf("[ws] Client connected for user %s from %s", userID, c.RemoteAddr())

	// Keep the connection alive by reading (and ignoring) messages from the client.
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ws] Unexpected close: %v", err)
			}
			break
		}
	}
}
