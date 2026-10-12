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
	authH := handler.NewAuthHandler(db, cfg)
	endpointH := handler.NewEndpointHandler(db)
	webhookH := handler.NewWebhookHandler(db, rc, mq, cfg)
	wsH := handler.NewWebSocketHandler(db, h, cfg)

	// Admin / API Key generation (can be called by dashboard / developers)
	app.Post("/api/admin/generate-key", authH.GenerateKey)

	// REST API routes.
	v1 := app.Group("/api/v1")
	{
		// Authentication & OAuth endpoints
		authGroup := v1.Group("/auth")
		{
			authGroup.Get("/config", authH.Config)
			authGroup.Get("/github", authH.LoginGitHub)
			authGroup.Get("/github/callback", authH.CallbackGitHub)
			authGroup.Get("/google", authH.LoginGoogle)
			authGroup.Get("/google/callback", authH.CallbackGoogle)
			authGroup.Post("/dev-login", authH.DevLogin)
			authGroup.Post("/keys", authH.GenerateKey)

			// Current authenticated user info
			authGroup.Get("/me", middleware.RequireAuth(db, cfg.JWTSecret), authH.Me)
		}

		// Protected endpoints (require API key or JWT authentication)
		protected := v1.Group("", middleware.RequireAuth(db, cfg.JWTSecret))
		{
			// Webhook endpoints ("Hooks")
			protected.Post("/endpoints", endpointH.Register)
			protected.Get("/endpoints", endpointH.List)
			protected.Delete("/endpoints/:id", endpointH.Delete)

			// Webhook sending and logs
			protected.Post("/webhooks/send", webhookH.Send)
			protected.Get("/logs", webhookH.GetLogs)
			protected.Get("/events/:id/deliveries", webhookH.GetDeliveries)
			protected.Post("/events/:id/replay", webhookH.Replay)
		}
	}

	// WebSocket route for live log streaming (token validated in UpgradeMiddleware).
	app.Use("/ws/logs", wsH.UpgradeMiddleware)
	app.Get("/ws/logs", websocket.New(wsH.Handle))

	// Health check.
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	return app
}
