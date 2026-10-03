package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"webhook-gateway/internal/hub"
)

// WebSocketHandler manages WebSocket upgrade and client lifecycle.
type WebSocketHandler struct {
	hub *hub.Hub
}

// NewWebSocketHandler creates a WebSocketHandler.
func NewWebSocketHandler(h *hub.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: h}
}

// UpgradeMiddleware checks that the request is a WebSocket upgrade request.
// Must be used as middleware before the WebSocket handler in Fiber.
func (h *WebSocketHandler) UpgradeMiddleware(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// Handle is the Fiber WebSocket handler for /ws/logs.
// Registers the client with the hub and blocks until disconnection.
func (h *WebSocketHandler) Handle(c *websocket.Conn) {
	h.hub.Register(c)
	defer h.hub.Unregister(c)

	log.Printf("[ws] Client connected from %s", c.RemoteAddr())

	// Keep the connection alive by reading (and ignoring) messages from the client.
	// Ping/pong is handled by the underlying library automatically.
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
