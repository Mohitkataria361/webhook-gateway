package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateAPIKey creates a raw API key prefixed with "sk_live_" and returns both the raw key and its SHA-256 hash.
func GenerateAPIKey() (rawKey string, keyHash string, err error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawKey = "sk_live_" + hex.EncodeToString(bytes)
	keyHash = HashAPIKey(rawKey)
	return rawKey, keyHash, nil
}

// HashAPIKey returns the hex-encoded SHA-256 hash of a raw API key.
func HashAPIKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}
