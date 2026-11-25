// Package ratelimit provides Redis-backed rate limiting using sliding window algorithm.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket is an in-process token bucket for fail-closed fallback.
// Used when Redis is unavailable for critical tiers (OTP request, verify, onboard).
type TokenBucket struct {
	mu       sync.Mutex
	capacity int
	tokens   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	key      string
}

// NewTokenBucket creates a new token bucket.
func NewTokenBucket(key string, capacity int, refillRatePerSecond float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity),
		refillRate: refillRatePerSecond,
		lastRefill: time.Now(),
		key:        key,
	}
}

// Allow checks if a request is allowed and consumes a token if so.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(float64(tb.capacity), tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Remaining returns the approximate number of tokens remaining.
func (tb *TokenBucket) Remaining() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokens := min(float64(tb.capacity), tb.tokens+elapsed*tb.refillRate)
	return int(tokens)
}

// FallbackLimiter provides in-process rate limiting for fail-closed tiers.
// It uses a sharded map of token buckets keyed by rate limit key.
type FallbackLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket
	capacity int
	refillRate float64
	shards int
}

// NewFallbackLimiter creates a new fallback limiter.
func NewFallbackLimiter(capacity int, refillRatePerSecond float64) *FallbackLimiter {
	return &FallbackLimiter{
		buckets:    make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRatePerSecond,
		shards:     16,
	}
}

// getOrCreateBucket gets or creates a token bucket for the given key.
func (fl *FallbackLimiter) getOrCreateBucket(key string) *TokenBucket {
	fl.mu.RLock()
	bucket, ok := fl.buckets[key]
	fl.mu.RUnlock()

	if ok {
		return bucket
	}

	fl.mu.Lock()
	defer fl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, ok := fl.buckets[key]; ok {
		return bucket
	}

	bucket = NewTokenBucket(key, fl.capacity, fl.refillRate)
	fl.buckets[key] = bucket
	return bucket
}

// Allow checks the rate limit using the fallback token bucket.
// Returns a Decision (always allowed=false when bucket exhausted).
func (fl *FallbackLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	bucket := fl.getOrCreateBucket(key)
	allowed := bucket.Allow()

	remaining := bucket.Remaining()
	if remaining < 0 {
		remaining = 0
	}

	retryAfter := time.Duration(0)
	if !allowed {
		retryAfter = time.Second // approximate
	}

	return Decision{
		Allowed:    allowed,
		Current:    int64(fl.capacity - remaining),
		Limit:      fl.capacity,
		Remaining:  int64(remaining),
		ResetAfter: window,
		RetryAfter: retryAfter,
	}, nil
}

// CleanupOldBuckets removes buckets that haven't been used recently.
// Should be called periodically to prevent memory leaks.
func (fl *FallbackLimiter) CleanupOldBuckets(maxAge time.Duration) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	now := time.Now()
	for key, bucket := range fl.buckets {
		bucket.mu.Lock()
		if now.Sub(bucket.lastRefill) > maxAge {
			delete(fl.buckets, key)
		}
		bucket.mu.Unlock()
	}
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}