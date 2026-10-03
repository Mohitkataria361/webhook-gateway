package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Sign computes an HMAC-SHA256 signature over payload using secretKey.
// Returns the hex-encoded digest prefixed with "sha256=" for compatibility
// with the X-Hub-Signature-256 header convention used by GitHub, Stripe, etc.
func Sign(secretKey string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(payload)
	digest := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("sha256=%s", digest)
}

// Verify checks whether the provided signature matches the recomputed one.
// Uses hmac.Equal to prevent timing attacks.
func Verify(secretKey string, payload []byte, signature string) bool {
	expected := Sign(secretKey, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}
