package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"webhook-gateway/internal/auth"
	"webhook-gateway/internal/config"
	"webhook-gateway/internal/models"
)

// AuthHandler handles authentication flows including OAuth and local dev login.
type AuthHandler struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(db *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

// Config returns the configuration status of auth providers.
func (h *AuthHandler) Config(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"github_enabled": h.cfg.GitHubClientID != "" && h.cfg.GitHubClientSecret != "",
		"google_enabled": h.cfg.GoogleClientID != "" && h.cfg.GoogleClientSecret != "",
		"dev_enabled":    true,
	})
}

// LoginGitHub redirects the user to GitHub's OAuth authorization page.
func (h *AuthHandler) LoginGitHub(c *fiber.Ctx) error {
	if h.cfg.GitHubClientID == "" || h.cfg.GitHubClientSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "GitHub OAuth is not configured on the server. Please set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET or use Dev Login.",
		})
	}

	state := generateRandomState()
	redirectURI := fmt.Sprintf("%s/api/v1/auth/github/callback", strings.TrimRight(h.cfg.APIBaseURL, "/"))
	authURL := auth.BuildGitHubAuthURL(h.cfg.GitHubClientID, redirectURI, state)
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

// CallbackGitHub handles the GitHub OAuth callback.
func (h *AuthHandler) CallbackGitHub(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		errDesc := c.Query("error_description", "Missing authorization code from GitHub")
		return h.redirectToDashboardWithError(c, errDesc)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	redirectURI := fmt.Sprintf("%s/api/v1/auth/github/callback", strings.TrimRight(h.cfg.APIBaseURL, "/"))
	accessToken, err := auth.ExchangeGitHubCode(ctx, h.cfg.GitHubClientID, h.cfg.GitHubClientSecret, code, redirectURI)
	if err != nil {
		log.Printf("[auth] GitHub code exchange failed: %v", err)
		return h.redirectToDashboardWithError(c, "GitHub code exchange failed: "+err.Error())
	}

	userInfo, err := auth.FetchGitHubUser(ctx, accessToken)
	if err != nil {
		log.Printf("[auth] Fetch GitHub user failed: %v", err)
		return h.redirectToDashboardWithError(c, "Failed to retrieve GitHub user details")
	}

	user, err := h.upsertUser(ctx, "github", userInfo)
	if err != nil {
		log.Printf("[auth] Upsert user failed: %v", err)
		return h.redirectToDashboardWithError(c, "Failed to save user account")
	}

	token, err := auth.GenerateToken(user.ID.String(), user.Email, user.Name, h.cfg.JWTSecret)
	if err != nil {
		log.Printf("[auth] Generate token failed: %v", err)
		return h.redirectToDashboardWithError(c, "Failed to create session token")
	}

	return h.redirectToDashboardWithToken(c, token)
}

// LoginGoogle redirects the user to Google's OAuth authorization page.
func (h *AuthHandler) LoginGoogle(c *fiber.Ctx) error {
	if h.cfg.GoogleClientID == "" || h.cfg.GoogleClientSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Google OAuth is not configured on the server. Please set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET or use Dev Login.",
		})
	}

	state := generateRandomState()
	redirectURI := fmt.Sprintf("%s/api/v1/auth/google/callback", strings.TrimRight(h.cfg.APIBaseURL, "/"))
	authURL := auth.BuildGoogleAuthURL(h.cfg.GoogleClientID, redirectURI, state)
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

// CallbackGoogle handles the Google OAuth callback.
func (h *AuthHandler) CallbackGoogle(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		errDesc := c.Query("error_description", "Missing authorization code from Google")
		return h.redirectToDashboardWithError(c, errDesc)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	redirectURI := fmt.Sprintf("%s/api/v1/auth/google/callback", strings.TrimRight(h.cfg.APIBaseURL, "/"))
	accessToken, err := auth.ExchangeGoogleCode(ctx, h.cfg.GoogleClientID, h.cfg.GoogleClientSecret, code, redirectURI)
	if err != nil {
		log.Printf("[auth] Google code exchange failed: %v", err)
		return h.redirectToDashboardWithError(c, "Google code exchange failed: "+err.Error())
	}

	userInfo, err := auth.FetchGoogleUser(ctx, accessToken)
	if err != nil {
		log.Printf("[auth] Fetch Google user failed: %v", err)
		return h.redirectToDashboardWithError(c, "Failed to retrieve Google user details")
	}

	user, err := h.upsertUser(ctx, "google", userInfo)
	if err != nil {
		log.Printf("[auth] Upsert user failed: %v", err)
		return h.redirectToDashboardWithError(c, "Failed to save user account")
	}

	token, err := auth.GenerateToken(user.ID.String(), user.Email, user.Name, h.cfg.JWTSecret)
	if err != nil {
		log.Printf("[auth] Generate token failed: %v", err)
		return h.redirectToDashboardWithError(c, "Failed to create session token")
	}

	return h.redirectToDashboardWithToken(c, token)
}

// DevLoginRequest payload for fast local authentication and testing.
type DevLoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

// DevLogin creates or signs in a developer/test user instantly without 3rd-party OAuth.
func (h *AuthHandler) DevLogin(c *fiber.Ctx) error {
	var req DevLoginRequest
	_ = c.BodyParser(&req)

	if req.Username == "" {
		req.Username = "alice"
	}
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	if req.Email == "" {
		req.Email = fmt.Sprintf("%s@example.com", req.Username)
	}
	if req.Name == "" {
		req.Name = strings.Title(req.Username)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	avatarURL := fmt.Sprintf("https://api.dicebear.com/7.x/identicon/svg?seed=%s", req.Username)
	userInfo := &auth.OAuthUserInfo{
		ProviderUserID: req.Username,
		Email:          req.Email,
		Name:           req.Name,
		AvatarURL:      avatarURL,
	}

	user, err := h.upsertUser(ctx, "dev", userInfo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create dev user: " + err.Error(),
		})
	}

	token, err := auth.GenerateToken(user.ID.String(), user.Email, user.Name, h.cfg.JWTSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user":  user,
	})
}

// Me returns the profile of the currently authenticated user.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id in token"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err = h.db.QueryRow(ctx, `
		SELECT id, email, name, provider, provider_user_id, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users
		WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Provider, &user.ProviderUserID, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(fiber.Map{"user": user})
}

// upsertUser finds existing user by provider+provider_user_id (or email), or creates a new one.
func (h *AuthHandler) upsertUser(ctx context.Context, provider string, info *auth.OAuthUserInfo) (*models.User, error) {
	var user models.User

	// Try find by provider & provider_user_id
	err := h.db.QueryRow(ctx, `
		SELECT id, email, name, provider, provider_user_id, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users
		WHERE provider = $1 AND provider_user_id = $2`,
		provider, info.ProviderUserID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Provider, &user.ProviderUserID, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err == nil {
		// Found existing user; update avatar/name/updated_at if changed
		_ = h.db.QueryRow(ctx, `
			UPDATE users
			SET name = $1, avatar_url = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
			RETURNING id, email, name, provider, provider_user_id, COALESCE(avatar_url, ''), created_at, updated_at`,
			info.Name, info.AvatarURL, user.ID,
		).Scan(&user.ID, &user.Email, &user.Name, &user.Provider, &user.ProviderUserID, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
		return &user, nil
	}

	// Try find by email (account linking across providers)
	err = h.db.QueryRow(ctx, `
		SELECT id, email, name, provider, provider_user_id, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users
		WHERE email = $1`,
		info.Email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Provider, &user.ProviderUserID, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err == nil {
		return &user, nil
	}

	// Insert new user
	err = h.db.QueryRow(ctx, `
		INSERT INTO users (email, name, provider, provider_user_id, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, provider, provider_user_id, COALESCE(avatar_url, ''), created_at, updated_at`,
		info.Email, info.Name, provider, info.ProviderUserID, info.AvatarURL,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Provider, &user.ProviderUserID, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

func (h *AuthHandler) redirectToDashboardWithToken(c *fiber.Ctx, token string) error {
	dest := fmt.Sprintf("%s?token=%s", strings.TrimRight(h.cfg.AppURL, "/"), url.QueryEscape(token))
	return c.Redirect(dest, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) redirectToDashboardWithError(c *fiber.Ctx, errMsg string) error {
	dest := fmt.Sprintf("%s?auth_error=%s", strings.TrimRight(h.cfg.AppURL, "/"), url.QueryEscape(errMsg))
	return c.Redirect(dest, fiber.StatusTemporaryRedirect)
}

func generateRandomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateKeyRequest allows optional custom user ID specification.
type GenerateKeyRequest struct {
	UserID string `json:"user_id,omitempty"`
}

// GenerateKeyResponse returns the newly generated API key.
type GenerateKeyResponse struct {
	APIKey    string    `json:"api_key"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// GenerateKey handles generation of B2B API keys.
// The raw key is returned once; the database only stores the SHA-256 hash.
func (h *AuthHandler) GenerateKey(c *fiber.Ctx) error {
	var req GenerateKeyRequest
	_ = c.BodyParser(&req)

	var uid uuid.UUID
	// 1. Check authenticated user context
	if authUID, ok := c.Locals("user_id").(string); ok && authUID != "" {
		parsed, err := uuid.Parse(authUID)
		if err == nil {
			uid = parsed
		}
	}

	// 2. Check request payload if context had none
	if uid == uuid.Nil && req.UserID != "" {
		parsed, err := uuid.Parse(req.UserID)
		if err == nil {
			uid = parsed
		}
	}

	// 3. Fallback: generate a new unique user UUID
	if uid == uuid.Nil {
		uid = uuid.New()
	}

	rawKey, keyHash, err := auth.GenerateAPIKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate API key: " + err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	_, err = h.db.Exec(ctx, `
		INSERT INTO api_keys (key_hash, user_id, created_at, is_active)
		VALUES ($1, $2, $3, TRUE)`,
		keyHash, uid, now,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to store API key in database: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(GenerateKeyResponse{
		APIKey:    rawKey,
		UserID:    uid.String(),
		CreatedAt: now,
		Message:   "Store this API key safely — it will never be displayed again.",
	})
}

