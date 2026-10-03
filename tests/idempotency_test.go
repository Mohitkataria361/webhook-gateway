package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestIdempotencySetNX tests the Redis SETNX idempotency pattern directly.
// Requires a live Redis instance at localhost:6379.
// To skip in CI without Redis: go test -run TestHMAC ./...
func TestIdempotency_SetNX_FirstCall_Succeeds(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	key := fmt.Sprintf("idempotency:test-%d", time.Now().UnixNano())

	// First call must return true (newly set).
	ok, err := rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if err != nil {
		t.Fatalf("SetNX error: %v", err)
	}
	if !ok {
		t.Error("expected first SetNX to return true (newly set)")
	}
}

func TestIdempotency_SetNX_DuplicateCall_ReturnsFalse(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	key := fmt.Sprintf("idempotency:dup-test-%d", time.Now().UnixNano())

	// First call sets the key.
	rdb.SetNX(ctx, key, "1", 24*time.Hour)

	// Second call with same key must return false (already exists).
	ok, err := rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if err != nil {
		t.Fatalf("SetNX error: %v", err)
	}
	if ok {
		t.Error("expected duplicate SetNX to return false")
	}
}

func TestIdempotency_SetNX_KeyExpires(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	key := fmt.Sprintf("idempotency:expire-test-%d", time.Now().UnixNano())

	// Set with 1-second TTL.
	rdb.SetNX(ctx, key, "1", 1*time.Second)

	// Verify TTL is set.
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL error: %v", err)
	}
	if ttl <= 0 {
		t.Error("expected key to have positive TTL")
	}

	// Wait for expiry.
	time.Sleep(2 * time.Second)

	// Key should be gone — SetNX should succeed again.
	ok, _ := rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if !ok {
		t.Error("expected key to have expired and SetNX to succeed")
	}
}
