package hub

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
)

// LogEvent is the structured payload broadcast to all connected dashboard clients.
type LogEvent struct {
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

// Hub maintains the set of active WebSocket clients and uses Redis Pub/Sub
// to broadcast messages across different processes (API vs Worker).
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
	rdb     *redis.Client
}

// New creates a Hub and starts its internal Redis subscription goroutine.
func New(rdb *redis.Client) *Hub {
	h := &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		rdb:     rdb,
	}
	go h.subscribe()
	return h
}

// Register adds a new WebSocket client connection.
func (h *Hub) Register(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	log.Printf("[hub] Client connected. Total clients: %d", len(h.clients))
}

// Unregister removes a WebSocket client connection.
func (h *Hub) Unregister(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	c.Close()
	log.Printf("[hub] Client disconnected. Total clients: %d", len(h.clients))
}

// Broadcast publishes the event to the Redis "ws:logs" channel instead of local memory.
// This allows the Worker process to send logs to the API process.
func (h *Hub) Broadcast(event LogEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[hub] JSON marshal error: %v", err)
		return
	}
	h.rdb.Publish(context.Background(), "ws:logs", data)
}

// subscribe listens to the Redis "ws:logs" channel and pushes messages to all connected WebSockets.
func (h *Hub) subscribe() {
	pubsub := h.rdb.Subscribe(context.Background(), "ws:logs")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		h.mu.RLock()
		for client := range h.clients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				log.Printf("[hub] Write error: %v", err)
			}
		}
		h.mu.RUnlock()
	}
}
