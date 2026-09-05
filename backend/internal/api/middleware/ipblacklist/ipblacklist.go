// Package ipblacklist provides a Redis-backed IP auto-blacklisting middleware
// that monitors client request patterns for security violations and
// automatically blocks IPs that exceed configured thresholds.
//
// Features:
//   - Tracks security violations (401, 429) per client IP using sliding windows
//   - Auto-blacklists IPs after threshold breaches (e.g., 5 violations in window)
//   - Instant 403 rejection for blacklisted IPs with canonical "access_denied" error
//   - Fail-open behavior: Redis errors during blacklist checks allow requests through
//   - Configurable violation types, thresholds, windows, and blacklist TTL
//
// Integration:
//
//	import "somotracker/backend/internal/api/middleware/ipblacklist"
//
//	blacklist := ipblacklist.NewIPBlacklistMiddleware(redisClient, logger, ipblacklist.Config{
//	    ViolationThreshold: 5,
//	    ViolationWindow:    15 * time.Minute,
//	    BlacklistTTL:       1 * time.Hour,
//	    ViolationCodes:     []int{fiber.StatusUnauthorized, fiber.StatusTooManyRequests},
//	})
//	app.Use(blacklist)
//
// The middleware should be placed early in the middleware chain (after request ID
// and CORS) so it can inspect response status codes from downstream handlers.
package ipblacklist

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Config holds the configuration for the IP blacklist middleware.
type Config struct {
	// ViolationThreshold is the number of security violations within the
	// ViolationWindow that triggers an auto-blacklist.
	// Default: 5
	ViolationThreshold int

	// ViolationWindow is the rolling time window for counting violations.
	// Default: 15 minutes
	ViolationWindow time.Duration

	// BlacklistTTL is how long a blacklisted IP remains blocked.
	// Default: 1 hour
	BlacklistTTL time.Duration

	// ViolationCodes are the HTTP status codes considered security violations.
	// Default: [401, 429]
	ViolationCodes []int

	// TrustedProxies is a list of CIDR ranges for trusted proxies.
	// IPs matching these will not be blacklisted.
	// Default: empty (no trusted proxies)
	TrustedProxies []string

	// SkipPaths are request paths that should not be tracked for violations
	// or blacklist checks (e.g., health checks).
	SkipPaths []string
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		ViolationThreshold: 5,
		ViolationWindow:    15 * time.Minute,
		BlacklistTTL:       1 * time.Hour,
		ViolationCodes:     []int{fiber.StatusUnauthorized, fiber.StatusTooManyRequests},
		TrustedProxies:     []string{},
		SkipPaths:          []string{"/health", "/livez", "/readyz"},
	}
}

// ipBlacklistMiddleware holds the state for the IP blacklist middleware.
type ipBlacklistMiddleware struct {
	client       *redis.Client
	logger       *zap.Logger
	config       Config
	violationSet map[int]bool
	skipPathSet  map[string]bool
}

// NewIPBlacklistMiddleware creates a new IP blacklist middleware handler.
//
// The middleware:
//  1. Checks if the client IP is blacklisted on every request (fail-open)
//  2. After the request completes, inspects the response status code
//  3. If it's a violation code, increments the violation counter for that IP
//  4. If violations exceed threshold, adds IP to blacklist with TTL
//
// Parameters:
//   - client: Redis client for blacklist storage and violation tracking
//   - logger: Zap logger for structured security event logging
//   - config: Configuration for thresholds, windows, and TTLs
//
// Returns a Fiber handler that can be registered with app.Use().
func NewIPBlacklistMiddleware(client *redis.Client, logger *zap.Logger, config Config) fiber.Handler {
	if config.ViolationThreshold <= 0 {
		config.ViolationThreshold = 5
	}
	if config.ViolationWindow <= 0 {
		config.ViolationWindow = 15 * time.Minute
	}
	if config.BlacklistTTL <= 0 {
		config.BlacklistTTL = 1 * time.Hour
	}
	if len(config.ViolationCodes) == 0 {
		config.ViolationCodes = []int{fiber.StatusUnauthorized, fiber.StatusTooManyRequests}
	}

	var middlewareLogger *zap.Logger
	if logger != nil {
		middlewareLogger = logger.With(zap.String("middleware", "ipblacklist"))
	} else {
		middlewareLogger = zap.NewNop()
	}

	m := &ipBlacklistMiddleware{
		client:       client,
		logger:       middlewareLogger,
		config:       config,
		violationSet: make(map[int]bool),
		skipPathSet:  make(map[string]bool),
	}

	for _, code := range config.ViolationCodes {
		m.violationSet[code] = true
	}
	for _, path := range config.SkipPaths {
		m.skipPathSet[path] = true
	}

	return m.handle
}

// handle is the main middleware handler function.
func (m *ipBlacklistMiddleware) handle(c fiber.Ctx) error {
	// Skip tracking for configured paths (health checks, etc.)
	if m.skipPathSet[c.Path()] {
		return c.Next()
	}

	// Extract client IP (respects X-Forwarded-For, X-Real-IP, CF-Connecting-IP via Fiber's c.IP())
	clientIP := c.IP()
	if clientIP == "" || clientIP == "unknown" {
		// Cannot determine IP, skip blacklist check but continue request
		return c.Next()
	}

	// Check if IP is in trusted proxies (skip blacklist)
	if m.isTrustedProxy(clientIP) {
		return c.Next()
	}

	// Check if IP is blacklisted (fail-open: Redis errors allow request)
	blacklisted, err := m.isBlacklisted(c.Context(), clientIP)
	if err != nil {
		// Fail-open: log error but allow request through
		m.logger.Warn("ipblacklist: redis error during blacklist check (fail-open)",
			zap.String("ip", clientIP),
			zap.Error(err),
			zap.String("request_id", c.Get("X-Request-ID")),
		)
		return c.Next()
	}

	if blacklisted {
		m.logger.Warn("ipblacklist: blocked blacklisted IP",
			zap.String("ip", clientIP),
			zap.String("request_id", c.Get("X-Request-ID")),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
		)

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    "access_denied",
			"message": "Access denied",
			"errors":  fiber.Map{},
		})
	}

	// Execute the request and capture the response status
	err = c.Next()

	// After request completes, check if response is a violation
	if m.isViolation(c.Response().StatusCode()) {
		m.recordViolation(c.Context(), clientIP, c.Response().StatusCode(), c.Get("X-Request-ID"))
	}

	return err
}

// isBlacklisted checks if an IP is in the Redis blacklist set.
// Uses fail-open: any Redis error returns (false, error) so caller can allow request.
func (m *ipBlacklistMiddleware) isBlacklisted(ctx context.Context, ip string) (bool, error) {
	if m.client == nil {
		return false, nil
	}

	key := blacklistKey(ip)
	exists, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return exists > 0, nil
}

// recordViolation increments the violation counter for an IP and
// blacklists if threshold is exceeded.
func (m *ipBlacklistMiddleware) recordViolation(ctx context.Context, ip string, statusCode int, requestID string) {
	if m.client == nil {
		return
	}

	violationKey := violationKey(ip)
	blacklistKey := blacklistKey(ip)

	// Use a pipeline for atomic check-and-increment
	pipe := m.client.Pipeline()

	// Increment violation counter
	incr := pipe.Incr(ctx, violationKey)

	// Set expiry on violation key if this is the first violation
	pipe.Expire(ctx, violationKey, m.config.ViolationWindow)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		m.logger.Error("ipblacklist: failed to record violation",
			zap.String("ip", ip),
			zap.Int("status_code", statusCode),
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return
	}

	violations := incr.Val()

	m.logger.Info("ipblacklist: security violation recorded",
		zap.String("ip", ip),
		zap.Int("status_code", statusCode),
		zap.Int64("violation_count", violations),
		zap.Int("threshold", m.config.ViolationThreshold),
		zap.String("request_id", requestID),
	)

	// Check if threshold exceeded
	if violations >= int64(m.config.ViolationThreshold) {
		// Add to blacklist with TTL
		err := m.client.Set(ctx, blacklistKey, "1", m.config.BlacklistTTL).Err()
		if err != nil {
			m.logger.Error("ipblacklist: failed to blacklist IP",
				zap.String("ip", ip),
				zap.Error(err),
				zap.String("request_id", requestID),
			)
			return
		}

		// Clean up violation counter since IP is now blacklisted
		_ = m.client.Del(ctx, violationKey).Err()

		m.logger.Warn("ipblacklist: IP auto-blacklisted",
			zap.String("ip", ip),
			zap.Int64("violation_count", violations),
			zap.Int("threshold", m.config.ViolationThreshold),
			zap.Duration("blacklist_ttl", m.config.BlacklistTTL),
			zap.String("request_id", requestID),
		)
	}
}

// isViolation checks if a status code is considered a security violation.
func (m *ipBlacklistMiddleware) isViolation(statusCode int) bool {
	return m.violationSet[statusCode]
}

// isTrustedProxy checks if an IP matches any trusted proxy CIDR.
func (m *ipBlacklistMiddleware) isTrustedProxy(ip string) bool {
	if len(m.config.TrustedProxies) == 0 {
		return false
	}
	// Simple string prefix match for common proxy IPs
	// For production, consider using net.IPNet for CIDR matching
	for _, proxy := range m.config.TrustedProxies {
		if strings.HasPrefix(ip, proxy) || ip == proxy {
			return true
		}
	}
	return false
}

// blacklistKey returns the Redis key for a blacklisted IP.
func blacklistKey(ip string) string {
	return "ip:blacklist:" + sanitizeKey(ip)
}

// violationKey returns the Redis key for tracking violations per IP.
func violationKey(ip string) string {
	return "ip:violations:" + sanitizeKey(ip)
}

// sanitizeKey replaces characters that could be problematic in Redis keys.
func sanitizeKey(ip string) string {
	// Replace colons (IPv6) and other special chars
	return strings.ReplaceAll(ip, ":", "_")
}

// GetBlacklistStatus retrieves the current blacklist status for an IP.
// Returns (isBlacklisted, remainingTTL, violationCount, error).
// Useful for admin dashboards or debugging.
func GetBlacklistStatus(ctx context.Context, client *redis.Client, ip string) (bool, time.Duration, int64, error) {
	if client == nil {
		return false, 0, 0, fmt.Errorf("client is nil")
	}

	pipe := client.Pipeline()
	blacklistExists := pipe.Exists(ctx, blacklistKey(ip))
	blacklistTTL := pipe.TTL(ctx, blacklistKey(ip))
	violationCount := pipe.Get(ctx, violationKey(ip))

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, 0, 0, fmt.Errorf("pipeline exec: %w", err)
	}

	isBlacklisted := blacklistExists.Val() > 0
	ttl := blacklistTTL.Val()
	var violations int64
	if violationCount.Err() == nil {
		violations, _ = strconv.ParseInt(violationCount.Val(), 10, 64)
	} else if violationCount.Err() != redis.Nil {
		violations = 0
	}

	return isBlacklisted, ttl, violations, nil
}

// RemoveFromBlacklist manually removes an IP from the blacklist.
// Returns error if Redis operation fails.
func RemoveFromBlacklist(ctx context.Context, client *redis.Client, ip string) error {
	if client == nil {
		return fmt.Errorf("client is nil")
	}
	return client.Del(ctx, blacklistKey(ip), violationKey(ip)).Err()
}
