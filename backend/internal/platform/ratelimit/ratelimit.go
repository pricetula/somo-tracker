package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter provides Redis-backed rate limiting with in-memory fallback.
type Limiter struct {
	client    *redis.Client
	hmacKey   []byte
	memFallback *memoryLimiter
}

// New creates a limiter with Redis and optional in-memory fallback.
func New(client *redis.Client, hmacKey string) *Limiter {
	var key []byte
	if hmacKey != "" {
		key = []byte(hmacKey)
	}
	return &Limiter{
		client:      client,
		hmacKey:     key,
		memFallback: newMemoryLimiter(),
	}
}

// AllowIP checks and increments an IP-based counter with optional tenant scoping.
// Returns true if request is allowed, false if limit exceeded.
func (l *Limiter) AllowIP(ctx context.Context, prefix, ip string, limit int, window time.Duration) (bool, error) {
	return l.allow(ctx, "ip", prefix, ip, "", limit, window)
}

// AllowEmail checks and increments an email-based counter using HMAC keyed hash.
// Returns true if request is allowed, false if limit exceeded.
func (l *Limiter) AllowEmail(ctx context.Context, prefix, email string, limit int, window time.Duration) (bool, error) {
	if l.hmacKey == nil {
		return false, fmt.Errorf("hmac key not configured")
	}
	hash := hmacEmail(email, l.hmacKey)
	return l.allow(ctx, "email", prefix, hash, "", limit, window)
}

// AllowTenantIP checks IP limit scoped to a tenant (prevents cross-tenant DoS).
func (l *Limiter) AllowTenantIP(ctx context.Context, prefix, tenantID, ip string, limit int, window time.Duration) (bool, error) {
	return l.allow(ctx, "ip", prefix, ip, tenantID, limit, window)
}

// AllowTenantEmail checks email limit scoped to a tenant.
func (l *Limiter) AllowTenantEmail(ctx context.Context, prefix, tenantID, email string, limit int, window time.Duration) (bool, error) {
	if l.hmacKey == nil {
		return false, fmt.Errorf("hmac key not configured")
	}
	hash := hmacEmail(email, l.hmacKey)
	return l.allow(ctx, "email", prefix, hash, tenantID, limit, window)
}

// AllowGlobal checks a global counter (e.g., email provider quota).
// Should be called AFTER per-IP and per-email checks.
func (l *Limiter) AllowGlobal(ctx context.Context, name string, limit int, window time.Duration) (bool, error) {
	return l.allow(ctx, "global", name, "", "", limit, window)
}

// AllowTenantGlobal checks a global counter scoped to tenant.
func (l *Limiter) AllowTenantGlobal(ctx context.Context, tenantID, name string, limit int, window time.Duration) (bool, error) {
	return l.allow(ctx, "global", name, "", tenantID, limit, window)
}

// Allow exposes in-memory fallback for testing.
// Only works when Redis client is nil (test mode).
func (l *Limiter) Allow(key string, limit int, window time.Duration) bool {
	return l.memFallback.Allow(key, limit, window)
}

// allow is the internal method that tries Redis first, falls back to memory.
func (l *Limiter) allow(ctx context.Context, kind, prefix, identifier, tenantID string, limit int, window time.Duration) (bool, error) {
	key := l.buildKey(kind, prefix, identifier, tenantID)

	// Try Redis first
	if l.client != nil {
		allowed, err := l.checkRedis(ctx, key, limit, window)
		if err == nil {
			return allowed, nil
		}
		// Redis error - fall through to memory
		// Log error in production
	}

	// Fallback to in-memory limiter
	return l.memFallback.Allow(key, limit, window), nil
}

// buildKey constructs the rate limit key.
func (l *Limiter) buildKey(kind, prefix, identifier, tenantID string) string {
	if tenantID != "" {
		return fmt.Sprintf("somotracker:ratelimit:%s:%s:tenant:%s:%s", kind, prefix, tenantID, identifier)
	}
	return fmt.Sprintf("somotracker:ratelimit:%s:%s:%s", kind, prefix, identifier)
}

// checkRedis performs atomic INCR + EXPIRE using Lua script.
func (l *Limiter) checkRedis(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	script := `
	local current = redis.call("INCR", KEYS[1])
	if tonumber(current) == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	if tonumber(current) > tonumber(ARGV[2]) then
		return 0
	else
		return 1
	end
	`
	res, err := l.client.Eval(ctx, script, []string{key}, int(window.Seconds()), limit).Result()
	if err != nil {
		return false, err
	}
	allowed, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected redis result")
	}
	return allowed == 1, nil
}

func hmacEmail(email string, key []byte) string {
	// Normalize email to lower case
	norm := ""
	for _, c := range email {
		if c >= 'A' && c <= 'Z' {
			norm += string(c + 32)
		} else {
			norm += string(c)
		}
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(norm))
	return hex.EncodeToString(mac.Sum(nil))
}

// memoryLimiter provides in-memory rate limiting as Redis fallback.
// Uses sliding window log for better accuracy than fixed window.
type memoryLimiter struct {
	mu    sync.Mutex
	store map[string][]time.Time // key -> list of request timestamps
}

func newMemoryLimiter() *memoryLimiter {
	return &memoryLimiter{
		store: make(map[string][]time.Time),
	}
}

// Allow checks if request is allowed under sliding window.
func (m *memoryLimiter) Allow(key string, limit int, window time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-window)

	// Get or create timestamp list
	times := m.store[key]
	if times == nil {
		times = make([]time.Time, 0, limit+1)
	}

	// Remove expired timestamps (sliding window)
	valid := times[:0]
	for _, t := range times {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	// Check limit
	if len(valid) >= limit {
		m.store[key] = valid // update with cleaned list
		return false
	}

	// Add current request
	valid = append(valid, now)
	m.store[key] = valid
	return true
}

// Cleanup removes expired entries periodically (call via background goroutine).
func (m *memoryLimiter) Cleanup(window time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-window)
	for key, times := range m.store {
		valid := times[:0]
		for _, t := range times {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(m.store, key)
		} else {
			m.store[key] = valid
		}
	}
}

// MemoryCleanup exposes the memory fallback cleanup for external callers.
func (l *Limiter) MemoryCleanup(window time.Duration) {
	l.memFallback.Cleanup(window)
}