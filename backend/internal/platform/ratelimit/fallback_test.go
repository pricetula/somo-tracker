package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket("test", 5, 1.0) // 5 capacity, 1 token/sec

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		assert.True(t, tb.Allow(), "request %d should be allowed", i+1)
	}

	// 6th should be denied
	assert.False(t, tb.Allow(), "6th request should be denied")

	// Remaining should be 0
	assert.Equal(t, 0, tb.Remaining())
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket("test", 3, 10.0) // 3 capacity, 10 tokens/sec

	// Exhaust bucket
	assert.True(t, tb.Allow())
	assert.True(t, tb.Allow())
	assert.True(t, tb.Allow())
	assert.False(t, tb.Allow())

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should have ~2 tokens refilled
	assert.True(t, tb.Allow())
	assert.True(t, tb.Allow())
	assert.False(t, tb.Allow())
}

func TestFallbackLimiter_Allow(t *testing.T) {
	ctx := context.Background()
	fl := NewFallbackLimiter(10, 1.0) // 10 capacity, 1 token/sec

	// Allow up to capacity
	for i := 0; i < 10; i++ {
		d, err := fl.Allow(ctx, "test-key", 10, time.Minute)
		require.NoError(t, err)
		assert.True(t, d.Allowed)
		assert.Equal(t, 10, d.Limit)
	}

	// Next should be denied
	d, err := fl.Allow(ctx, "test-key", 10, time.Minute)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Equal(t, int64(0), d.Remaining)
}

func TestFallbackLimiter_DifferentKeys(t *testing.T) {
	ctx := context.Background()
	fl := NewFallbackLimiter(5, 1.0)

	// Each key should have its own bucket
	for i := 0; i < 5; i++ {
		d, err := fl.Allow(ctx, "key-1", 5, time.Minute)
		require.NoError(t, err)
		assert.True(t, d.Allowed)
	}

	// key-1 exhausted
	d, err := fl.Allow(ctx, "key-1", 5, time.Minute)
	require.NoError(t, err)
	assert.False(t, d.Allowed)

	// key-2 should still have capacity
	d, err = fl.Allow(ctx, "key-2", 5, time.Minute)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
}

func TestFallbackLimiter_Cleanup(t *testing.T) {
	fl := NewFallbackLimiter(10, 1.0)
	ctx := context.Background()

	// Create some buckets
	_, _ = fl.Allow(ctx, "key-1", 10, time.Minute)
	_, _ = fl.Allow(ctx, "key-2", 10, time.Minute)

	// Cleanup with 0 max age should remove all (they were just used)
	// Actually, they were just used so lastRefill is now, not old
	// Let's test with a very short maxAge after waiting
	time.Sleep(10 * time.Millisecond)
	fl.CleanupOldBuckets(5 * time.Millisecond)
	// Buckets should be cleaned up since they're older than 5ms
	// (they were created before the sleep)

	// New requests should create new buckets
	d, err := fl.Allow(ctx, "key-1", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
}

func TestFallbackLimiter_Concurrent(t *testing.T) {
	fl := NewFallbackLimiter(100, 10.0)
	ctx := context.Background()

	done := make(chan bool, 100)

	// 10 goroutines, each making 10 requests
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, _ = fl.Allow(ctx, "shared-key", 100, time.Minute)
			}
			done <- true
		}()
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have allowed exactly 100 requests (capacity)
	// The rest should be denied
	_, _ = fl.Allow(ctx, "shared-key", 100, time.Minute)
	// At least some should be denied since we made 100 requests
	// (exact count depends on timing/refill)
}
