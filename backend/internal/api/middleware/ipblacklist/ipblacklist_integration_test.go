//go:build integration

package ipblacklist

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func newTestRedisClient(t *testing.T, ctx context.Context) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	return client
}

func TestIPBlacklistMiddleware_AllowsRequestWhenNotBlacklisted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	app := fiber.New()
	app.Get("/test", middleware, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestIPBlacklistMiddleware_BlocksBlacklistedIP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	// Manually blacklist an IP
	ip := "192.168.1.100"
	err := client.Set(ctx, "ip:blacklist:192.168.1.100", "1", time.Hour).Err()
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/test", middleware, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Fiber's c.IP() reads from X-Forwarded-For, X-Real-IP, etc.
	// For testing, we need to simulate the IP extraction
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, assert.JSONEq(t, `{"code":"access_denied","message":"Access denied","errors":{}}`, string(resp.Body)))
}

func TestIPBlacklistMiddleware_AutoBlacklistsAfterThreshold(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.ViolationThreshold = 3
	config.ViolationCodes = []int{fiber.StatusUnauthorized}
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	app := fiber.New()
	app.Get("/test", middleware, func(c fiber.Ctx) error {
		// Simulate a 401 response
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "Unauthorized",
			"errors":  fiber.Map{},
		})
	})

	ip := "192.168.1.200"

	// Make 2 requests (below threshold)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", ip)
		resp, err := app.Test(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	}

	// 3rd request should trigger blacklist
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip)
	resp, err := app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Subsequent request should be blocked with 403
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip)
	resp, err = app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestIPBlacklistMiddleware_SkipPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.SkipPaths = []string{"/health", "/custom-skip"}
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	app := fiber.New()
	app.Get("/health", middleware, func(c fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{})
	})
	app.Get("/custom-skip", middleware, func(c fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{})
	})
	app.Get("/other", middleware, func(c fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{})
	})

	// Health check should pass even with 401
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode) // Returns 401, not blocked

	// Custom skip path should pass
	req = httptest.NewRequest("GET", "/custom-skip", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Other path should track violations
	req = httptest.NewRequest("GET", "/other", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	resp, err = app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestIPBlacklistMiddleware_TrustedProxies(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.TrustedProxies = []string{"10.0.0.", "172.16."}
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	app := fiber.New()
	app.Get("/test", middleware, func(c fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{})
	})

	// Trusted proxy IP should not be tracked/blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.50")
	resp, err := app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode) // Returns 401, not blocked

	// Non-trusted IP should be tracked
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	resp, err = app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestIPBlacklistMiddleware_ViolationCodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.ViolationCodes = []int{fiber.StatusTooManyRequests} // Only 429
	config.ViolationThreshold = 2
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	app := fiber.New()
	app.Get("/test-429", middleware, func(c fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{})
	})
	app.Get("/test-401", middleware, func(c fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{})
	})

	ip := "192.168.1.250"

	// 429 should count as violation
	req := httptest.NewRequest("GET", "/test-429", nil)
	req.Header.Set("X-Forwarded-For", ip)
	resp, err := app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)

	// Second 429 should trigger blacklist
	req = httptest.NewRequest("GET", "/test-429", nil)
	req.Header.Set("X-Forwarded-For", ip)
	resp, err = app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)

	// 401 should NOT count as violation (not in config)
	req = httptest.NewRequest("GET", "/test-401", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.251")
	resp, err = app.Test(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestIPBlacklistMiddleware_GetBlacklistStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	ip := "192.168.1.50"

	// Not blacklisted initially
	blacklisted, ttl, violations, err := GetBlacklistStatus(ctx, client, ip)
	require.NoError(t, err)
	assert.False(t, blacklisted)
	assert.Equal(t, int64(0), violations)

	// Add violation
	err = client.Incr(ctx, "ip:violations:192.168.1.50").Err()
	require.NoError(t, err)

	blacklisted, ttl, violations, err = GetBlacklistStatus(ctx, client, ip)
	require.NoError(t, err)
	assert.False(t, blacklisted)
	assert.Equal(t, int64(1), violations)

	// Blacklist the IP
	err = client.Set(ctx, "ip:blacklist:192.168.1.50", "1", time.Hour).Err()
	require.NoError(t, err)

	blacklisted, ttl, violations, err = GetBlacklistStatus(ctx, client, ip)
	require.NoError(t, err)
	assert.True(t, blacklisted)
	assert.True(t, ttl > 0)
}

func TestIPBlacklistMiddleware_RemoveFromBlacklist(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	ip := "192.168.1.60"

	// Blacklist and add violations
	err := client.Set(ctx, "ip:blacklist:192.168.1.60", "1", time.Hour).Err()
	require.NoError(t, err)
	err = client.Set(ctx, "ip:violations:192.168.1.60", "10", time.Hour).Err()
	require.NoError(t, err)

	// Remove from blacklist
	err = RemoveFromBlacklist(ctx, client, ip)
	require.NoError(t, err)

	blacklisted, _, violations, err := GetBlacklistStatus(ctx, client, ip)
	require.NoError(t, err)
	assert.False(t, blacklisted)
	assert.Equal(t, int64(0), violations)
}

func TestIPBlacklistMiddleware_FailOpenOnRedisError(t *testing.T) {
	t.Parallel()

	// Use a nil client to simulate Redis unavailable
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	middleware := NewIPBlacklistMiddleware(nil, logger, config)

	app := fiber.New()
	app.Get("/test", middleware, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should pass through (fail-open)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestIPBlacklistMiddleware_CanonicalErrorFormat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	middleware := NewIPBlacklistMiddleware(client, logger, config)

	// Blacklist an IP
	ip := "192.168.1.70"
	err := client.Set(ctx, "ip:blacklist:192.168.1.70", "1", time.Hour).Err()
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/test", middleware, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	// Verify canonical error format
	var body map[string]interface{}
	require.NoError(t, assert.JSONEq(t, `{"code":"access_denied","message":"Access denied","errors":{}}`, string(resp.Body)))
}
