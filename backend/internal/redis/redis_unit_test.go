package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

func newTestConfig(_ *testing.T) *config.Config {
	return &config.Config{
		RedisURL: "redis://localhost:6379/0",
	}
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
