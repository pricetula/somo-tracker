package testcontainers

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	tc "github.com/testcontainers/testcontainers-go"
	redisModule "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer holds a testcontainers Redis instance
type RedisContainer struct {
	Container tc.Container
	Client    *redis.Client
	addr      string
}

// SetupRedis starts a Redis container using testcontainers-go
// and returns a RedisContainer with a connected client.
// The container is automatically terminated when the test completes (via ryuk).
func SetupRedis(t *testing.T) *RedisContainer {
	ctx := context.Background()

	rdContainer, err := redisModule.Run(ctx,
		"redis:7-alpine",
		tc.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "failed to start redis container")

	// Get connection address
	host, err := rdContainer.Host(ctx)
	require.NoError(t, err, "failed to get redis host")

	port, err := rdContainer.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err, "failed to get redis port")

	addr := host + ":" + port.Port()

	// Create client
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Verify connection
	require.Eventually(t, func() bool {
		return client.Ping(ctx).Err() == nil
	}, 10*time.Second, 100*time.Millisecond)

	rc := &RedisContainer{
		Container: rdContainer,
		Client:    client,
		addr:      addr,
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		if rc.Client != nil {
			rc.Client.Close()
		}
	})

	return rc
}

// Addr returns the Redis address (host:port)
func (rc *RedisContainer) Addr() string {
	return rc.addr
}

// Terminate manually terminates the container (usually not needed due to ryuk).
func (rc *RedisContainer) Terminate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if rc.Client != nil {
		rc.Client.Close()
	}
	if rc.Container != nil {
		if err := rc.Container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	}
}
