package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"webhook-gateway/internal/api"
	"webhook-gateway/internal/config"
	"webhook-gateway/internal/db"
	"webhook-gateway/internal/hub"
	"webhook-gateway/internal/queue"
	redisclient "webhook-gateway/internal/redis"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[api] Starting Webhook Gateway API Server")

	// --- Load configuration ---
	cfg := config.Load()

	// --- Connect to PostgreSQL ---
	pool, err := db.NewPool(cfg)
	if err != nil {
		log.Fatalf("[api] Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	// --- Run database migrations ---
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	migrationsDir := filepath.Join(projectRoot, "internal", "db", "migrations")
	if err := db.RunMigrations(pool, migrationsDir); err != nil {
		log.Fatalf("[api] Migration failed: %v", err)
	}

	// --- Connect to Redis ---
	rc, err := redisclient.New(cfg)
	if err != nil {
		log.Fatalf("[api] Failed to connect to Redis: %v", err)
	}
	log.Println("[api] Connected to Redis")

	// --- Connect to RabbitMQ ---
	mq, err := queue.New(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("[api] Failed to connect to RabbitMQ: %v", err)
	}
	defer mq.Close()
	log.Println("[api] Connected to RabbitMQ")

	// --- Create WebSocket Hub with Redis Pub/Sub ---
	h := hub.New(rc.Raw())

	// --- Set up and start Fiber ---
	app := api.SetupRouter(pool, rc, mq, h, cfg)

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[api] Shutting down server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("[api] Shutdown error: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%s", cfg.APIPort)
	log.Printf("[api] Listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Printf("[api] Server stopped: %v", err)
	}
}
