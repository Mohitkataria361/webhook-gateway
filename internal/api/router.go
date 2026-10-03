package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"webhook-gateway/internal/api/handler"
	"webhook-gateway/internal/api/middleware"
	"webhook-gateway/internal/config"
	"webhook-gateway/internal/hub"
	"webhook-gateway/internal/queue"
	redisclient "webhook-gateway/internal/redis"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupRouter builds and returns a configured Fiber app with all routes.
func SetupRouter(
	db *pgxpool.Pool,
	rc *redisclient.Client,
	mq *queue.RabbitMQ,
	h *hub.Hub,
	cfg *config.Config,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "Webhook Gateway",
		DisableStartupMessage: false,
	})

	// Global middleware.
	app.Use(middleware.Logger())
	app.Use(middleware.CORS())

	// Instantiate handlers.
	endpointH := handler.NewEndpointHandler(db)
	webhookH := handler.NewWebhookHandler(db, rc, mq, cfg)
	wsH := handler.NewWebSocketHandler(h)

	// REST API routes.
	v1 := app.Group("/api/v1")
	{
		// Endpoint registration.
		v1.Post("/endpoints", endpointH.Register)
		v1.Get("/endpoints", endpointH.List)

		// Webhook event ingestion.
		v1.Post("/webhooks/send", webhookH.Send)

		// Delivery history + replay.
		v1.Get("/events/:id/deliveries", webhookH.GetDeliveries)
		v1.Post("/events/:id/replay", webhookH.Replay)
	}

	// WebSocket route for live log streaming.
	app.Use("/ws/logs", wsH.UpgradeMiddleware)
	app.Get("/ws/logs", websocket.New(wsH.Handle))

	// Health check.
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	return app
}
