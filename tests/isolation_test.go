package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"webhook-gateway/internal/api/middleware"
	"webhook-gateway/internal/auth"
)

func TestAuthMiddleware_Enforcement(t *testing.T) {
	jwtSecret := "super-secure-jwt-secret-testing-32b"
	app := fiber.New()

	app.Get("/protected", middleware.RequireAuth(nil, jwtSecret), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		email := c.Locals("email").(string)
		return c.JSON(fiber.Map{"user_id": userID, "email": email})
	})

	// 1. Unauthenticated request should return 401
	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	// 2. Request with invalid token should return 401
	req = httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-junk-token")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	// 3. Request with valid token for Alice should succeed
	aliceID := "550e8400-e29b-41d4-a716-446655440001"
	token, err := auth.GenerateToken(aliceID, "alice@test.com", "Alice", jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req = httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}
