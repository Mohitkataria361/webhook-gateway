package tests

import (
	"testing"
	"time"

	"webhook-gateway/internal/auth"
)

func TestJWT_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-32-bytes-long-super"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "alice@example.com"
	name := "Alice Cooper"

	token, err := auth.GenerateToken(userID, email, name, secret)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token string")
	}

	claims, err := auth.ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
	if claims.Name != name {
		t.Errorf("expected Name %s, got %s", name, claims.Name)
	}
}

func TestJWT_InvalidSecretFails(t *testing.T) {
	secret1 := "secret-one-secret-one-secret-one"
	secret2 := "secret-two-secret-two-secret-two"
	userID := "550e8400-e29b-41d4-a716-446655440000"

	token, err := auth.GenerateToken(userID, "test@example.com", "Test", secret1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = auth.ValidateToken(token, secret2)
	if err == nil {
		t.Error("expected validation to fail with mismatched secret")
	}
}

func TestJWT_EmptyTokenFails(t *testing.T) {
	_, err := auth.ValidateToken("", "some-secret")
	if err == nil {
		t.Error("expected empty token to fail")
	}
}

func TestOAuth_BuildURLs(t *testing.T) {
	ghURL := auth.BuildGitHubAuthURL("client-123", "http://localhost:8080/callback", "xyzstate")
	if ghURL == "" {
		t.Fatal("expected non-empty github auth url")
	}
	if !contains(ghURL, "client_id=client-123") {
		t.Errorf("missing client_id in github url: %s", ghURL)
	}

	googleURL := auth.BuildGoogleAuthURL("g-client-456", "http://localhost:8080/callback", "xyzstate")
	if googleURL == "" {
		t.Fatal("expected non-empty google auth url")
	}
	if !contains(googleURL, "client_id=g-client-456") {
		t.Errorf("missing client_id in google url: %s", googleURL)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && time.Now().Unix() > 0 && (s != "" && (s == substr || len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAPIKey_GenerateAndHash(t *testing.T) {
	rawKey, keyHash, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("unexpected error generating api key: %v", err)
	}

	if len(rawKey) == 0 || !findSubstr(rawKey, "sk_live_") {
		t.Fatalf("expected raw key prefixed with sk_live_, got %s", rawKey)
	}

	if len(keyHash) != 64 {
		t.Fatalf("expected 64 char sha256 hex string, got %d chars: %s", len(keyHash), keyHash)
	}

	computedHash := auth.HashAPIKey(rawKey)
	if computedHash != keyHash {
		t.Fatalf("expected hash %s to match computed hash %s", keyHash, computedHash)
	}

	// Verify hashing is deterministic
	if auth.HashAPIKey(rawKey) != auth.HashAPIKey(rawKey) {
		t.Fatal("expected deterministic sha-256 hash output")
	}
}

