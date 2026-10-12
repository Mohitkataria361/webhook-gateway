package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"webhook-gateway/internal/auth"
)

// RequireAuth validates an API Key (via SHA-256 database lookup) or a JWT token
// passed in the Authorization header (Bearer <token>), query parameter (?token=... or ?api_key=...), or cookie.
// On success, sets user_id in fiber context Locals.
func RequireAuth(db *pgxpool.Pool, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenStr string

		// 1. Check Authorization: Bearer <token>
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		}

		// 2. Check query param ?token=... or ?api_key=...
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			tokenStr = c.Query("api_key")
		}

		// 3. Check cookie "auth_token"
		if tokenStr == "" {
			tokenStr = c.Cookies("auth_token")
		}

		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Missing API Key or token. Please provide 'Authorization: Bearer <API_KEY>'",
			})
		}

		// Check if it's an API Key (or lookup in api_keys table if DB pool is provided)
		if db != nil {
			keyHash := auth.HashAPIKey(tokenStr)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var userID uuid.UUID
			err := db.QueryRow(ctx,
				"SELECT user_id FROM api_keys WHERE key_hash = $1 AND is_active = TRUE",
				keyHash,
			).Scan(&userID)

			if err == nil {
				c.Locals("user_id", userID.String())
				c.Locals("auth_type", "api_key")
				return c.Next()
			}
		}

		// Check if it's a valid JWT
		if jwtSecret != "" {
			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err == nil {
				c.Locals("user_id", claims.UserID)
				c.Locals("email", claims.Email)
				c.Locals("name", claims.Name)
				c.Locals("auth_type", "jwt")
				return c.Next()
			}
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "Invalid or expired API Key / token",
		})
	}
}
