package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisClient implements the redis.Client interface subset used by Limiter.
type mockRedisClient struct {
	incrementFunc func(ctx context.Context, key string, ttl time.Duration) (int64, error)
	getFunc       func(ctx context.Context, key string) (int64, error)
}

func (m *mockRedisClient) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return m.incrementFunc(ctx, key, ttl)
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (int64, error) {
	return m.getFunc(ctx, key)
}

// Ensure mockRedisClient satisfies the interface used by Limiter.
var _ interface {
	Increment(context.Context, string, time.Duration) (int64, error)
	Get(context.Context, string) (int64, error)
} = (*mockRedisClient)(nil)

func TestAllow_UnderLimit(t *testing.T) {
	ctx := context.Background()

	l := New(&mockRedisClient{
		incrementFunc: func(ctx context.Context, key string, ttl time.Duration) (int64, error) {
			return 1, nil
		},
	})

	allowed, count, err := l.Allow(ctx, "test:key", 5, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(1), count)
}

func TestAllow_AtLimit(t *testing.T) {
	ctx := context.Background()

	l := New(&mockRedisClient{
		incrementFunc: func(ctx context.Context, key string, ttl time.Duration) (int64, error) {
			return 5, nil
		},
	})

	allowed, count, err := l.Allow(ctx, "test:key", 5, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed) // 5 <= 5
	assert.Equal(t, int64(5), count)
}

func TestAllow_OverLimit(t *testing.T) {
	ctx := context.Background()

	l := New(&mockRedisClient{
		incrementFunc: func(ctx context.Context, key string, ttl time.Duration) (int64, error) {
			return 6, nil
		},
	})

	allowed, count, err := l.Allow(ctx, "test:key", 5, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(6), count)
}

func TestAllow_IncrementError(t *testing.T) {
	ctx := context.Background()

	l := New(&mockRedisClient{
		incrementFunc: func(ctx context.Context, key string, ttl time.Duration) (int64, error) {
			return 0, errors.New("redis down")
		},
	})

	_, _, err := l.Allow(ctx, "test:key", 5, time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis down")
}

func TestRemaining(t *testing.T) {
	ctx := context.Background()

	l := New(&mockRedisClient{
		getFunc: func(ctx context.Context, key string) (int64, error) {
			return 3, nil
		},
	})

	count, err := l.Remaining(ctx, "test:key")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestRemaining_Nil(t *testing.T) {
	ctx := context.Background()

	l := New(&mockRedisClient{
		getFunc: func(ctx context.Context, key string) (int64, error) {
			return 0, nil
		},
	})

	count, err := l.Remaining(ctx, "test:key")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
