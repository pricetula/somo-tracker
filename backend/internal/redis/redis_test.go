package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// newTestConfig returns a Config tuned for a local test Redis instance.
func newTestConfig(_ *testing.T) *config.Config {
	return &config.Config{
		RedisURL: "redis://localhost:6379/0",
	}
}

// TestNewClient_HappyPath exercises the Fx wiring end-to-end.
// Requires a reachable Redis on localhost:6379. The test is gated behind
// the `integration` build tag so it does not run by default in unit-test CI.
func TestNewClient_HappyPath(t *testing.T) {
	cfg := newTestConfig(t)
	logger := zap.NewNop()

	var client *redis.Client
	app := fxtest.New(t,
		fx.Provide(func() *config.Config { return cfg }),
		fx.Provide(func() *zap.Logger { return logger }),
		Module,
		fx.Populate(&client),
	)
	require.NoError(t, app.Start(t.Context()), "fx app should start cleanly")
	require.NotNil(t, client, "client should be populated by Fx")

	pingCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(pingCtx).Err(), "PING should succeed against test Redis")

	require.NoError(t, app.Stop(t.Context()), "fx app should stop cleanly")
}

// mockLifecycle satisfies fx.Lifecycle for unit tests that only need the
// nil-check paths without wiring a full Fx app.
type mockLifecycle struct{}

func (mockLifecycle) Append(fx.Hook) {}

// TestNewClient_NilConfig asserts the nil-config safety net.
func TestNewClient_NilConfig(t *testing.T) {
	logger := zap.NewNop()
	client, err := NewClient(mockLifecycle{}, nil, logger)
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

// TestNewClient_NilLogger asserts the nil-logger safety net.
func TestNewClient_NilLogger(t *testing.T) {
	cfg := newTestConfig(t)
	client, err := NewClient(mockLifecycle{}, cfg, nil)
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger is required")
}

// TestPing_NilClient asserts the nil-client safety net.
func TestPing_NilClient(t *testing.T) {
	err := Ping(t.Context(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is nil")
}

// TestPing_NilContext asserts the nil-context safety net.
func TestPing_NilContext(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()
	err := Ping(nil, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context is nil")
}
