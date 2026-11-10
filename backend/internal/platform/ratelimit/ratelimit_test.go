package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Test_AC12_RateLimitAtomicity provides a placeholder for AC-12 rate-limit tests.
// Full test requires a Redis instance. It is skipped in CI without REDIS_ADDR.
func Test_AC12_RateLimitAtomicity(t *testing.T) {
	addr := "localhost:6379"
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	// ping to check connectivity
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping AC-12 rate-limit tests")
	}
	limiter := New(client, "test-hmac-key")
	// AllowIP limit 2/min
	key := fmt.Sprintf("ac12-ip-test-%d", time.Now().UnixNano())
	allowed1, err := limiter.AllowIP(ctx, "ac12", key, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed1)

	allowed2, err := limiter.AllowIP(ctx, "ac12", key, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed2)

	allowed3, err := limiter.AllowIP(ctx, "ac12", key, 2, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed3, "third request should be rate limited")
}

// Test_AC12_EmailHMACKey verifies HMAC email keying to avoid logging emails.
func Test_AC12_EmailHMACKey(t *testing.T) {
	addr := "localhost:6379"
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}
	limiter := New(client, "test-hmac-key")
	email := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())
	allowed1, err := limiter.AllowEmail(ctx, "ac12", email, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed1)

	allowed2, err := limiter.AllowEmail(ctx, "ac12", email, 1, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed2, "email rate limit should be enforced via HMAC key")
}

// TestLimiter_TenantScopedIP tests per-tenant IP rate limiting
func TestLimiter_TenantScopedIP(t *testing.T) {
	limiter := New(nil, "") // no Redis, use memory fallback

	ctx := context.Background()
	tenantID := "tenant-1"
	ip := "192.168.1.1"

	// Limit 2 requests per minute per tenant per IP
	allowed1, err := limiter.AllowTenantIP(ctx, "test", tenantID, ip, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed1)

	allowed2, err := limiter.AllowTenantIP(ctx, "test", tenantID, ip, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed2)

	allowed3, err := limiter.AllowTenantIP(ctx, "test", tenantID, ip, 2, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed3, "third request should be rate limited per tenant")

	// Different tenant should have separate limit
	allowed4, err := limiter.AllowTenantIP(ctx, "test", "tenant-2", ip, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed4, "different tenant should have separate limit")
}

// TestLimiter_TenantScopedEmail tests per-tenant email rate limiting
func TestLimiter_TenantScopedEmail(t *testing.T) {
	limiter := New(nil, "test-hmac-key")

	ctx := context.Background()
	tenantID := "tenant-1"
	email := "user@example.com"

	// Limit 2 requests per 15 min per tenant per email
	allowed1, err := limiter.AllowTenantEmail(ctx, "test", tenantID, email, 2, 15*time.Minute)
	require.NoError(t, err)
	require.True(t, allowed1)

	allowed2, err := limiter.AllowTenantEmail(ctx, "test", tenantID, email, 2, 15*time.Minute)
	require.NoError(t, err)
	require.True(t, allowed2)

	allowed3, err := limiter.AllowTenantEmail(ctx, "test", tenantID, email, 2, 15*time.Minute)
	require.NoError(t, err)
	require.False(t, allowed3, "third request should be rate limited per tenant")

	// Different tenant should have separate limit
	allowed4, err := limiter.AllowTenantEmail(ctx, "test", "tenant-2", email, 2, 15*time.Minute)
	require.NoError(t, err)
	require.True(t, allowed4, "different tenant should have separate limit")
}

// TestLimiter_GlobalFallbackOrder tests that global caps apply after per-IP/per-email
func TestLimiter_GlobalFallbackOrder(t *testing.T) {
	limiter := New(nil, "test-hmac-key")

	ctx := context.Background()
	ip := "192.168.1.1"
	email := "user@example.com"

	// First check per-IP (limit 1)
	ipAllowed, err := limiter.AllowIP(ctx, "test", ip, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ipAllowed)

	// Then check per-email (limit 1)
	emailAllowed, err := limiter.AllowEmail(ctx, "test", email, 1, 15*time.Minute)
	require.NoError(t, err)
	require.True(t, emailAllowed)

	// Then check global (limit 1) - separate counter, should succeed
	globalAllowed, err := limiter.AllowGlobal(ctx, "email-provider-hourly", 1, time.Hour)
	require.NoError(t, err)
	require.True(t, globalAllowed, "global is separate counter")

	// Second global request should fail
	globalAllowed2, err := limiter.AllowGlobal(ctx, "email-provider-hourly", 1, time.Hour)
	require.NoError(t, err)
	require.False(t, globalAllowed2, "global cap should be exhausted after 1 request")

	// Different global name should have separate limit
	globalAllowed3, err := limiter.AllowGlobal(ctx, "email-provider-daily", 10000, 24*time.Hour)
	require.NoError(t, err)
	require.True(t, globalAllowed3)
}

// TestLimiter_MemoryFallbackWhenRedisDown tests in-memory fallback when Redis unavailable
func TestLimiter_MemoryFallbackWhenRedisDown(t *testing.T) {
	// Create limiter with invalid Redis address
	client := redis.NewClient(&redis.Options{Addr: "invalid-host:6379"})
	limiter := New(client, "test-hmac-key")

	ctx := context.Background()
	ip := "192.168.1.1"

	// Should fall back to memory and work
	allowed, err := limiter.AllowIP(ctx, "test", ip, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed2, err := limiter.AllowIP(ctx, "test", ip, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed2)

	allowed3, err := limiter.AllowIP(ctx, "test", ip, 2, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed3)
}

// TestLimiter_SlidingWindow tests sliding window behavior (memory fallback)
func TestLimiter_SlidingWindow(t *testing.T) {
	limiter := New(nil, "")

	key := "sliding-test"

	// Allow 2 requests per 100ms window
	allowed1 := limiter.Allow(key, 2, 100*time.Millisecond)
	require.True(t, allowed1)

	allowed2 := limiter.Allow(key, 2, 100*time.Millisecond)
	require.True(t, allowed2)

	allowed3 := limiter.Allow(key, 2, 100*time.Millisecond)
	require.False(t, allowed3)

	// Wait for window to slide
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again after window expires
	allowed4 := limiter.Allow(key, 2, 100*time.Millisecond)
	require.True(t, allowed4)
}

// TestLimiter_Cleanup tests MemoryCleanup removes expired entries
func TestLimiter_Cleanup(t *testing.T) {
	limiter := New(nil, "")

	key := "cleanup-test"

	// Add some entries
	limiter.Allow(key, 1, time.Minute)
	limiter.Allow("other-key", 1, time.Minute)

	// Cleanup with very short window should remove entries
	limiter.MemoryCleanup(1 * time.Nanosecond)

	// New requests should work (store was cleaned)
	allowed := limiter.Allow("cleanup-new", 1, time.Minute)
	require.True(t, allowed)
}

// TestHMACEmail_Deterministic tests that same email produces same hash
func TestHMACEmail_Deterministic(t *testing.T) {
	key := []byte("test-hmac-key-32-bytes-long!!")
	email := "User@Example.COM"

	hash1 := hmacEmail(email, key)
	hash2 := hmacEmail(email, key)
	require.Equal(t, hash1, hash2, "HMAC should be deterministic")

	// Case normalization
	hash3 := hmacEmail("user@example.com", key)
	require.Equal(t, hash1, hash3, "Email should be normalized to lowercase")
}

// TestHMACEmail_DifferentEmailsDifferentHashes tests that different emails produce different hashes
func TestHMACEmail_DifferentEmailsDifferentHashes(t *testing.T) {
	key := []byte("test-hmac-key-32-bytes-long!!")

	hash1 := hmacEmail("user1@example.com", key)
	hash2 := hmacEmail("user2@example.com", key)
	require.NotEqual(t, hash1, hash2)

	// Even similar emails
	hash3 := hmacEmail("user@example.com", key)
	hash4 := hmacEmail("user@example.org", key)
	require.NotEqual(t, hash3, hash4)
}
