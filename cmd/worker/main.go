package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"webhook-gateway/internal/config"
	"webhook-gateway/internal/db"
	"webhook-gateway/internal/hub"
	"webhook-gateway/internal/queue"
	redisclient "webhook-gateway/internal/redis"
	"webhook-gateway/internal/worker"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[worker] Starting Webhook Gateway Worker")

	cfg := config.Load()

	// --- Connect to PostgreSQL ---
	pool, err := db.NewPool(cfg)
	if err != nil {
		log.Fatalf("[worker] PostgreSQL connection failed: %v", err)
	}
	defer pool.Close()

	// --- Connect to Redis ---
	rc, err := redisclient.New(cfg)
	if err != nil {
		log.Fatalf("[worker] Redis connection failed: %v", err)
	}

	// --- Connect to RabbitMQ ---
	mq, err := queue.New(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("[worker] RabbitMQ connection failed: %v", err)
	}
	defer mq.Close()

	// The worker uses the Hub's Redis Pub/Sub to broadcast logs back to the API server.
	h := hub.New(rc.Raw())

	// Create cancellable context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("[worker] Received shutdown signal")
		cancel()
	}()

	// Start the worker pool (blocks until ctx is cancelled).
	pool_ := worker.NewPool(cfg.WorkerConcurrency, pool, rc, mq, h)
	if err := pool_.Start(ctx); err != nil {
		log.Fatalf("[worker] Pool error: %v", err)
	}

	log.Println("[worker] Graceful shutdown complete")
}
