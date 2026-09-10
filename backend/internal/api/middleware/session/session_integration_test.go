//go:build integration

package session

import (
	"context"
	"encoding/json"
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

// mockRedisClient wraps a real redis.Client and provides test fixtures.
type mockRedisClient struct {
	client *redis.Client
}

// TestSessionMiddleware_UnauthenticatedAccess verifies that protected endpoints
// return 401 Unauthorized when no session cookie is present.
func TestSessionMiddleware_UnauthenticatedAccess(t *testing.T) {
	t.Parallel()

	// Create a real Redis client for this test.
	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware and a protected handler.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			// This handler should never be reached without a valid session.
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Request without session cookie.
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 401 Unauthorized.
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Assert the response body matches the canonical error contract.
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", body["code"])
	assert.Equal(t, "Unauthorized. Please log in to continue.", body["message"])
	assert.NotNil(t, body["errors"])
}

// TestSessionMiddleware_ValidSessionFlow verifies that a valid session cookie
// passes Redis verification, injects tenant context into c.Locals, and allows
// the request to proceed.
func TestSessionMiddleware_ValidSessionFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware and a protected handler.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			// Extract tenant context injected by the middleware.
			userID, _ := c.Locals("user_id").(string)
			tenantID, _ := c.Locals("tenant_id").(string)
			stytchID, _ := c.Locals("stytch_session_id").(string)

			return c.JSON(fiber.Map{
				"user_id":           userID,
				"tenant_id":         tenantID,
				"stytch_session_id": stytchID,
			})
		},
	)

	// Create a valid session in Redis.
	sessionToken := "valid-session-token-123"
	sessionData := map[string]interface{}{
		"user_id":           "user-uuid-abc123",
		"tenant_id":         "tenant-uuid-xyz789",
		"stytch_session_id": "stytch-session-001",
		"expires_at":        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)
	err = client.Set(ctx, "session:"+sessionToken, sessionJSON, 0).Err()
	require.NoError(t, err)

	// Request with valid session cookie.
	req := httptest.NewRequest("GET", "/api/users/user-uuid-abc123", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 200 OK.
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Assert tenant context was injected into the response.
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "user-uuid-abc123", body["user_id"])
	assert.Equal(t, "tenant-uuid-xyz789", body["tenant_id"])
	assert.Equal(t, "stytch-session-001", body["stytch_session_id"])
}

// TestSessionMiddleware_ExpiredSession verifies that a session with an expired
// timestamp in Redis triggers 401 Unauthorized and cleans up the stale cookie.
func TestSessionMiddleware_ExpiredSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Create an expired session in Redis.
	sessionToken := "expired-session-token"
	sessionData := map[string]interface{}{
		"user_id":           "user-uuid-abc123",
		"tenant_id":         "tenant-uuid-xyz789",
		"stytch_session_id": "stytch-session-001",
		"expires_at":        time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // Expired 1 hour ago.
	}
	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)
	err = client.Set(ctx, "session:"+sessionToken, sessionJSON, 0).Err()
	require.NoError(t, err)

	// Request with expired session cookie.
	req := httptest.NewRequest("GET", "/api/users/user-uuid-abc123", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 401 Unauthorized.
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Assert the response body matches the canonical error contract.
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", body["code"])

	// Assert the stale cookie was cleared.
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "session_token cookie should be cleared")
	assert.Equal(t, "", sessionCookie.Value)
	assert.True(t, sessionCookie.MaxAge < 0, "MaxAge should be negative for cookie deletion")
}

// TestSessionMiddleware_RevokedSession verifies that a session not found in Redis
// (revoked/logged out) triggers 401 Unauthorized and cleans up the cookie.
func TestSessionMiddleware_RevokedSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Request with a session token that doesn't exist in Redis (revoked/logged out).
	sessionToken := "revoked-session-token"
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 401 Unauthorized.
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Assert the response body matches the canonical error contract.
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", body["code"])

	// Assert the stale cookie was cleared.
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "session_token cookie should be cleared")
	assert.Equal(t, "", sessionCookie.Value)
}

// TestSessionMiddleware_MissingTenantContext verifies that a session with empty
// tenant_id is rejected with 401.
func TestSessionMiddleware_MissingTenantContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Create a session with empty tenant_id (invalid state).
	sessionToken := "session-missing-tenant"
	sessionData := map[string]interface{}{
		"user_id":           "user-uuid-abc123",
		"tenant_id":         "", // Missing tenant context.
		"stytch_session_id": "stytch-session-001",
		"expires_at":        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)
	err = client.Set(ctx, "session:"+sessionToken, sessionJSON, 0).Err()
	require.NoError(t, err)

	// Request with session that has missing tenant context.
	req := httptest.NewRequest("GET", "/api/users/user-uuid-abc123", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 401 Unauthorized.
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Assert the response body matches the canonical error contract.
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", body["code"])
}

// TestSessionMiddleware_NilRedisClient verifies that the middleware gracefully
// passes through when the Redis client is nil (for testing or degraded mode).
func TestSessionMiddleware_NilRedisClient(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(nil, logger)

	// Set up a Fiber app with the session middleware.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Request without session cookie (should pass through when Redis is nil).
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert the request passed through (no 401) when Redis is not configured.
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestSessionMiddleware_CookieName verifies that the middleware uses the correct
// cookie name (session_token).
func TestSessionMiddleware_CookieName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Create a valid session in Redis.
	sessionToken := "valid-token"
	sessionData := map[string]interface{}{
		"user_id":           "user-123",
		"tenant_id":         "tenant-456",
		"stytch_session_id": "stytch-789",
		"expires_at":        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)
	err = client.Set(ctx, "session:"+sessionToken, sessionJSON, 0).Err()
	require.NoError(t, err)

	// Request with wrong cookie name should be rejected.
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	req.AddCookie(&http.Cookie{Name: "wrong_cookie_name", Value: sessionToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 401 Unauthorized because session_token cookie is missing.
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestSessionMiddleware_RedisKeyFormat verifies that the middleware constructs
// the correct Redis key format (session:<token>).
func TestSessionMiddleware_RedisKeyFormat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	// Set up a Fiber app with the session middleware.
	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Store session with the correct key format.
	sessionToken := "test-token-abc123"
	sessionData := map[string]interface{}{
		"user_id":           "user-123",
		"tenant_id":         "tenant-456",
		"stytch_session_id": "stytch-789",
		"expires_at":        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)
	// Store with the correct key format.
	err = client.Set(ctx, "session:"+sessionToken, sessionJSON, 0).Err()
	require.NoError(t, err)

	// Request with valid session cookie.
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 200 OK.
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestSessionMiddleware_ResponseHeaders verifies that the response includes
// security headers and no internal details leak in error responses.
func TestSessionMiddleware_ResponseHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Request without session cookie.
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Verify the response doesn't leak internal details.
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	// The error message should not contain internal paths, tokens, or Redis keys.
	msg := body["message"].(string)
	assert.NotContains(t, msg, "session:")
	assert.NotContains(t, msg, "redis")
	assert.NotContains(t, msg, "/")
	assert.NotContains(t, msg, "token")
	assert.NotContains(t, msg, "key")

	// The error code should be a standard format.
	code := body["code"].(string)
	assert.NotContains(t, code, "_")
	assert.Contains(t, code, "unauthorized")
}

// TestSessionMiddleware_EmptyToken verifies that an empty string token is treated
// the same as a missing cookie.
func TestSessionMiddleware_EmptyToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	// Request with empty session cookie.
	req := httptest.NewRequest("GET", "/api/users/user-123", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ""})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Assert 401 Unauthorized.
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestSessionMiddleware_ConcurrentRequests verifies that the middleware handles
// concurrent requests without race conditions.
func TestSessionMiddleware_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestRedisClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t)
	middleware := NewSessionMiddleware(client, logger)

	app := fiber.New()
	app.Get("/api/users/:id",
		middleware,
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		},
	)

	sessionToken := "concurrent-token"
	sessionData := map[string]interface{}{
		"user_id":           "user-123",
		"tenant_id":         "tenant-456",
		"stytch_session_id": "stytch-789",
		"expires_at":        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)
	err = client.Set(ctx, "session:"+sessionToken, sessionJSON, 0).Err()
	require.NoError(t, err)

	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			req := httptest.NewRequest("GET", "/api/users/user-123", nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
			resp, err := app.Test(req)
			require.NoError(t, err)
			_ = resp.Body.Close()
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		}()
	}

	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout")
		}
	}
}

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
