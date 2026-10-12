package hub

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
)

// LogEvent is the structured payload broadcast to connected dashboard clients.
type LogEvent struct {
	UserID          string  `json:"user_id,omitempty"`
	EventID         string  `json:"event_id"`
	EndpointID      string  `json:"endpoint_id"`
	TargetURL       string  `json:"target_url"`
	AttemptNumber   int     `json:"attempt_number"`
	HTTPStatus      *int    `json:"http_status,omitempty"`
	Status          string  `json:"status"`
	ExecutionTimeMS *int    `json:"execution_time_ms,omitempty"`
	ResponseBody    *string `json:"response_body,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
}

// Hub maintains the set of active WebSocket clients mapped to their user IDs,
// using Redis Pub/Sub to broadcast messages across API and Worker processes.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]string // conn -> userID
	rdb     *redis.Client
}

// New creates a Hub and starts its internal Redis subscription goroutine.
func New(rdb *redis.Client) *Hub {
	h := &Hub{
		clients: make(map[*websocket.Conn]string),
		rdb:     rdb,
	}
	go h.subscribe()
	return h
}

// Register adds a new WebSocket client connection associated with a user ID.
func (h *Hub) Register(c *websocket.Conn, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = userID
	log.Printf("[hub] Client connected for user %s. Total clients: %d", userID, len(h.clients))
}

// Unregister removes a WebSocket client connection.
func (h *Hub) Unregister(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	userID := h.clients[c]
	delete(h.clients, c)
	c.Close()
	log.Printf("[hub] Client disconnected for user %s. Total clients: %d", userID, len(h.clients))
}

// Broadcast publishes the event to the Redis "ws:logs" channel.
func (h *Hub) Broadcast(event LogEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[hub] JSON marshal error: %v", err)
		return
	}
	h.rdb.Publish(context.Background(), "ws:logs", data)
}

// subscribe listens to the Redis "ws:logs" channel and pushes messages ONLY
// to connected WebSockets belonging to the matching user.
func (h *Hub) subscribe() {
	pubsub := h.rdb.Subscribe(context.Background(), "ws:logs")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var event LogEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("[hub] Failed to decode log event payload: %v", err)
			continue
		}

		h.mu.RLock()
		for client, clientUserID := range h.clients {
			// If event has a UserID, deliver strictly to clients belonging to that user.
			if event.UserID != "" && clientUserID != "" && clientUserID != event.UserID {
				continue
			}
			if err := client.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				log.Printf("[hub] Write error: %v", err)
			}
		}
		h.mu.RUnlock()
	}
}
