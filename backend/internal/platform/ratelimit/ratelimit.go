// Package ratelimit provides Redis-backed rate limiting using atomic counters.
package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// counterStore is the subset of the Redis client used for rate limiting.
type counterStore interface {
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Get(ctx context.Context, key string) (int64, error)
}

// Limiter checks rate limits using Redis atomic counters.
type Limiter struct {
	store counterStore
}

// New creates a rate limiter backed by a counter store.
func New(store counterStore) *Limiter {
	return &Limiter{store: store}
}

// Allow checks if a request is within the rate limit for the given key.
// It increments the counter and returns whether the request is allowed,
// the current count, and any error.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int64, error) {
	count, err := l.store.Increment(ctx, key, window)
	if err != nil {
		return false, 0, fmt.Errorf("increment rate limit counter: %w", err)
	}
	return count <= int64(limit), count, nil
}

// Remaining returns the number of requests remaining for the given key.
func (l *Limiter) Remaining(ctx context.Context, key string) (int64, error) {
	return l.store.Get(ctx, key)
}
