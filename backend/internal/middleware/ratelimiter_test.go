package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_BotScenarios(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = rdb.Close() }()

	t.Run("Scenario 1: Concurrency Burst - 50 parallel requests within 1ms", func(t *testing.T) {
		app := fiber.New()

		// Allow max 5 requests per minute
		app.Use(NewRateLimiter(rdb, RateLimiterConfig{
			Limit:  5,
			Window: 1 * time.Minute,
			Prefix: "bot_burst",
			KeyLookup: func(c *fiber.Ctx) string {
				return c.IP()
			},
		}))

		app.Post("/api/v1/action", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		var (
			wg          sync.WaitGroup
			successCnt  int32
			rateLimited int32
			totalReqs   = 50
		)

		wg.Add(totalReqs)
		for i := 0; i < totalReqs; i++ {
			go func() {
				defer wg.Done()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/action", nil)
				resp, err := app.Test(req, -1) // -1 disables timeout for local testing
				if err != nil {
					return
				}

				switch resp.StatusCode {
				case http.StatusOK:
					atomic.AddInt32(&successCnt, 1)
				case http.StatusTooManyRequests:
					atomic.AddInt32(&rateLimited, 1)
				}
			}()
		}
		wg.Wait()

		// Verify atomic consistency
		assert.Equal(t, int32(5), successCnt, "Exactly 5 requests should pass")
		assert.Equal(t, int32(45), rateLimited, "Remaining 45 requests should be blocked with 429")
	})

	t.Run("Scenario 2: IP Hopping - Bot changes IP but keeps same User ID (Tier 2 Catch)", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ProxyHeader: fiber.HeaderXForwardedFor,
		})

		// 1. Mock Auth Middleware MUST run first to populate c.Locals("user_id")
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user_bot_123")
			return c.Next()
		})

		// 2. Tier 2 Rate Limiter runs second and safely reads c.Locals("user_id")
		app.Use(NewRateLimiter(rdb, RateLimiterConfig{
			Limit:  3,
			Window: 1 * time.Minute,
			Prefix: "bot_user_tier",
			KeyLookup: func(c *fiber.Ctx) string {
				if userID, ok := c.Locals("user_id").(string); ok {
					return userID
				}
				return ""
			},
		}))

		app.Post("/api/v1/resource", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		// Bot makes requests rotating IPs every time
		ips := []string{"192.168.1.1", "10.0.0.5", "172.16.0.2", "8.8.8.8"}

		for i, ip := range ips {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/resource", nil)
			req.Header.Set("X-Forwarded-For", ip)

			resp, err := app.Test(req)
			require.NoError(t, err)

			if i < 3 {
				assert.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("Request %d from IP %s should pass", i+1, ip))
			} else {
				// 4th request must be caught by Tier 2 despite new IP!
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "4th request must be blocked by User ID limiter")
			}
		}
	})

	t.Run("Scenario 3: Sliding Window Precision - Fast recovery test", func(t *testing.T) {
		app := fiber.New()

		window := 200 * time.Millisecond
		app.Use(NewRateLimiter(rdb, RateLimiterConfig{
			Limit:  2,
			Window: window,
			Prefix: "sliding_window_test",
			KeyLookup: func(c *fiber.Ctx) string {
				return c.IP()
			},
		}))

		app.Get("/api/v1/data", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		// 1. Send 2 requests (reaches limit)
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
			resp, _ := app.Test(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// 2. 3rd request blocked immediately
		reqBlocked := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
		respBlocked, _ := app.Test(reqBlocked)
		assert.Equal(t, http.StatusTooManyRequests, respBlocked.StatusCode)

		// 3. Fast-forward time past window expiration
		mr.FastForward(window + 10*time.Millisecond)

		// 4. Request should now succeed again
		reqRecovered := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
		respRecovered, _ := app.Test(reqRecovered)
		assert.Equal(t, http.StatusOK, respRecovered.StatusCode)
	})
}

func TestRateLimiter_FailsClosedOnRedisError(t *testing.T) {
	// Create a Redis client pointing to a closed miniredis instance
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	app := fiber.New()
	app.Use(NewRateLimiter(rdb, RateLimiterConfig{
		Limit:     10,
		Window:    1 * time.Minute,
		Prefix:    "fail_closed",
		KeyLookup: func(c *fiber.Ctx) string { return c.IP() },
	}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// Shut down Redis to simulate outage
	mr.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// With fail-closed, Redis errors must result in 503, not 200
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
