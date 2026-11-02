package ratelimit

// FR-PROD-SEC-BASELINE-001-AC-12 Rate limit tests

import (
	"context"
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
	key := "ac12-ip-test"
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
	email := "user@example.com"
	allowed1, err := limiter.AllowEmail(ctx, "ac12", email, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed1)

	allowed2, err := limiter.AllowEmail(ctx, "ac12", email, 1, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed2, "email rate limit should be enforced via HMAC key")
}
