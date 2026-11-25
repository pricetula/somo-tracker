// Package ratelimit provides Redis-backed rate limiting using sliding window algorithm.
package ratelimit

import (
	"context"
	"time"
)

// Decision represents the result of a rate limit check.
type Decision struct {
	Allowed    bool
	Current    int64
	Limit      int
	Remaining  int64
	ResetAfter time.Duration
	RetryAfter time.Duration
}

// Limiter is the core rate limiting interface.
type Limiter interface {
	// Allow checks the rate limit for a key and returns a Decision.
	// The key should be pre-built by KeyBuilder (HMAC email, normalized IP, etc.)
	Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error)
}

// KeyBuilder constructs hierarchical rate limit keys.
type KeyBuilder interface {
	// UserKey builds a key for authenticated user (highest priority).
	UserKey(userID string) string
	// ChallengeKey builds a key for OTP challenge ID.
	ChallengeKey(challengeID string) string
	// EmailKey builds a key for normalized email (HMAC-SHA256).
	EmailKey(normalizedEmail string) string
	// IPKey builds a key for normalized IP (/32 IPv4, /64 IPv6).
	IPKey(normalizedIP string) string
	// GlobalKey builds the global circuit breaker key.
	GlobalKey() string
	// DomainKey builds a key for email domain velocity limiting.
	DomainKey(domain string) string
	// TierPrefix returns the tier prefix for key namespacing.
	TierPrefix(tier string) string
}

// Store is the subset of Redis operations needed for rate limiting.
type Store interface {
	EvalScript(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	ScriptLoad(ctx context.Context, script string) (string, error)
	ScriptEvalSHA(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error)
}