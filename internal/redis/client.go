package redisclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"webhook-gateway/internal/config"
)

// Client wraps the go-redis client with domain-specific helpers.
type Client struct {
	rdb *redis.Client
	cfg *config.Config
}

// New creates and pings a new Redis client.
func New(cfg *config.Config) (*Client, error) {
	opts := &redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	}

	// Upstash requires TLS/SSL
	if strings.Contains(cfg.RedisAddr, "upstash.io") {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{rdb: rdb, cfg: cfg}, nil
}

// -----------------------------------------------------------------------
// Idempotency
// -----------------------------------------------------------------------

// idempotencyKey returns the Redis key for a given client idempotency key.
func idempotencyKey(key string) string {
	return fmt.Sprintf("idempotency:%s", key)
}

// SetIdempotency atomically sets the key with a 24-hour TTL using SETNX.
// Returns (true, nil) if the key was newly set (first time).
// Returns (false, nil) if the key already exists (duplicate request).
func (c *Client) SetIdempotency(ctx context.Context, key string) (bool, error) {
	ok, err := c.rdb.SetNX(ctx, idempotencyKey(key), "1", 24*time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("redis SETNX idempotency: %w", err)
	}
	return ok, nil
}

// -----------------------------------------------------------------------
// Circuit Breaker — Sliding Window Counter
// -----------------------------------------------------------------------

const (
	cbStateOpen   = "OPEN"
	cbStateClosed = "CLOSED"
)

func cbStateKey(endpointID string) string {
	return fmt.Sprintf("cb:state:%s", endpointID)
}

func cbSuccessKey(endpointID string) string {
	return fmt.Sprintf("cb:success:%s", endpointID)
}

func cbFailureKey(endpointID string) string {
	return fmt.Sprintf("cb:failure:%s", endpointID)
}

// IsCircuitOpen returns true if the circuit breaker for endpointID is OPEN.
func (c *Client) IsCircuitOpen(ctx context.Context, endpointID string) (bool, error) {
	state, err := c.rdb.Get(ctx, cbStateKey(endpointID)).Result()
	if err == redis.Nil {
		return false, nil // Key absent means CLOSED
	}
	if err != nil {
		return false, fmt.Errorf("redis GET circuit state: %w", err)
	}
	return state == cbStateOpen, nil
}

// RecordSuccess increments the success counter in the sliding window.
func (c *Client) RecordSuccess(ctx context.Context, endpointID string) error {
	pipe := c.rdb.Pipeline()
	pipe.Incr(ctx, cbSuccessKey(endpointID))
	pipe.Expire(ctx, cbSuccessKey(endpointID), 10*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// RecordFailure increments the failure counter and evaluates whether
// the circuit breaker should trip to OPEN state.
func (c *Client) RecordFailure(ctx context.Context, endpointID string) error {
	pipe := c.rdb.Pipeline()
	pipe.Incr(ctx, cbFailureKey(endpointID))
	pipe.Expire(ctx, cbFailureKey(endpointID), 10*time.Minute)
	results, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline failure counter: %w", err)
	}

	failures := results[0].(*redis.IntCmd).Val()

	// Fetch successes to compute total requests in window.
	successes, err := c.rdb.Get(ctx, cbSuccessKey(endpointID)).Int64()
	if err == redis.Nil {
		successes = 0
	} else if err != nil {
		return err
	}

	total := failures + successes
	threshold := int64(c.cfg.CBThreshold)
	window := int64(c.cfg.CBWindow)

	// Only evaluate once we have enough data in the window.
	if total >= window {
		failurePct := (failures * 100) / total
		if failurePct >= threshold {
			openDuration := time.Duration(c.cfg.CBOpenDurationMin) * time.Minute
			if err := c.rdb.Set(ctx, cbStateKey(endpointID), cbStateOpen, openDuration).Err(); err != nil {
				return fmt.Errorf("redis SET circuit OPEN: %w", err)
			}
			// Reset counters after tripping.
			c.rdb.Del(ctx, cbSuccessKey(endpointID), cbFailureKey(endpointID))
		}
	}
	return nil
}

// Raw returns the underlying *redis.Client for advanced operations.
func (c *Client) Raw() *redis.Client {
	return c.rdb
}
