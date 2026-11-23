//go:build integration

package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"somotracker/internal/platform/config"
)

// TestStartup_AppCreatesSuccessfully verifies the full app startup path:
//
//	1. Connects to a real PostgreSQL 18 database
//	2. Runs goose migrations
//	3. Connects to a real Redis instance and PINGs it
//	4. Returns a fully constructed App (no "failed to create app" error)
//
// This directly tests the code path that previously failed with:
//   failed to create app err="connect redis: ping redis: dial tcp [::1]:6379: connect: connection refused"
func TestStartup_AppCreatesSuccessfully(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping startup integration test in short mode")
	}

	ctx := context.Background()

	// --- PostgreSQL 18 (matches docker-compose.yml) ---
	pgContainer, err := postgres.Run(ctx,
		"postgres:18",
		postgres.WithDatabase("somotracker"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "failed to start PostgreSQL container")
	t.Cleanup(func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	})

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get PostgreSQL connection string")

	// --- Redis 7 (matches docker-compose.yml) ---
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(15 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err, "failed to start Redis container")
	t.Cleanup(func() {
		require.NoError(t, redisContainer.Terminate(ctx))
	})

	redisHostPort, err := redisContainer.Endpoint(ctx, "6379/tcp")
	require.NoError(t, err, "failed to get Redis endpoint")
	redisURL := "redis://" + redisHostPort

	// --- Config pointing at the real containers ---
	cfg := &config.Config{
		AppEnv:            "development",
		LogLevel:          "debug",
		ServerPort:        "3030",
		DatabaseURL:       pgDSN,
		RedisURL:          redisURL,
		SessionCookieName: "session",
		BrevoAPIKey:       "test-brevo-key",
		BrevoSenderEmail:  "noreply@somotracker.com",
		BrevoSenderName:   "Somotracker",
		EmailHMACKey:      "test-hmac-key-32-bytes-long!!",
		CORSOrigins:       "*",
	}

	// --- This is the exact startup path that failed before ---
	// app.New connects to Postgres, runs migrations, then connects to Redis.
	a, err := New(cfg)
	require.NoError(t, err, "failed to create app")
	require.NotNil(t, a)

	// --- Verify /readyz dependencies are reachable ---
	// DB ping
	require.NoError(t, a.dbPool.Ping(ctx), "database not reachable after startup")

	// Redis ping
	require.NoError(t, a.redis.Client.Ping(ctx).Err(), "redis not reachable after startup")

	// --- Graceful shutdown ---
	require.NoError(t, a.Shutdown(ctx), "failed to shut down app cleanly")
}
