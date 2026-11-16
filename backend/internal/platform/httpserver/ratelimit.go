package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/ratelimit"
)

// RateLimitMiddleware provides rate limiting for Fiber routes based on IP, email, and tenant context.
// It wraps internal/platform/ratelimit and exposes helper methods and Fiber handlers.
type RateLimitMiddleware struct {
	limiter *ratelimit.Limiter
	cfg     *config.Config
}

// NewRateLimitMiddleware creates a new middleware instance wrapping the limiter.
// All public rate limits are driven by cfg (env-configurable, see platform/config).
func NewRateLimitMiddleware(limiter *ratelimit.Limiter, cfg *config.Config) *RateLimitMiddleware {
	return &RateLimitMiddleware{limiter: limiter, cfg: cfg}
}

// PublicRateLimit enforces public rate limits on unauthenticated endpoints:
//   - Per-IP limit for all public routes
//   - Per-endpoint limits: magic-link, token-exchange, logout, provision
//
// Email-based and provider-quota limits are enforced in the identity handlers
// (where the JSON body/email is already available) to avoid double counting.
// Best used on public API groups before handler execution.
func (rl *RateLimitMiddleware) PublicRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		path := c.Path()

		// Per-IP limits by endpoint
		switch path {
		case "/api/v1/auth/request-magic-link":
			allowed, retryAfter := rl.checkLimit(c.Context(), "magic-link", ip, rl.cfg.RateLimitMagicLinkIPPerMin, time.Minute)
			if !allowed {
				return rl.rateLimited(c, retryAfter)
			}

		case "/api/v1/auth/authenticate-token":
			allowed, retryAfter := rl.checkLimit(c.Context(), "token-exchange", ip, rl.cfg.RateLimitTokenExchangeIPPerMin, time.Minute)
			if !allowed {
				return rl.rateLimited(c, retryAfter)
			}

		case "/api/v1/auth/logout":
			allowed, retryAfter := rl.checkLimit(c.Context(), "logout", ip, rl.cfg.RateLimitLogoutIPPerMin, time.Minute)
			if !allowed {
				return rl.rateLimited(c, retryAfter)
			}

		case "/api/v1/onboarding/complete":
			allowed, retryAfter := rl.checkLimit(c.Context(), "provision", ip, rl.cfg.RateLimitProvisionIPPerMin, time.Minute)
			if !allowed {
				return rl.rateLimited(c, retryAfter)
			}

		default:
			// Generic public IP limit for other unauthenticated routes
			allowed, retryAfter := rl.checkLimit(c.Context(), "public", ip, rl.cfg.RateLimitPublicIPPerMin, time.Minute)
			if !allowed {
				return rl.rateLimited(c, retryAfter)
			}
		}

		return c.Next()
	}
}

// checkLimit runs a single IP-based rate limit check. On limiter error it fails
// open (the in-memory fallback already absorbs Redis outages) so a transient
// store failure cannot knock out public endpoints.
func (rl *RateLimitMiddleware) checkLimit(ctx context.Context, prefix, ip string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration) {
	allowed, err := rl.limiter.AllowIP(ctx, prefix, ip, limit, window)
	if err != nil {
		slog.Warn("public rate limit check failed, failing open", "err", err)
		return true, 0
	}
	if !allowed {
		retryAfter = window
	}
	return allowed, retryAfter
}

// rateLimited returns a 429 with a Retry-After hint when available.
func (rl *RateLimitMiddleware) rateLimited(c fiber.Ctx, retryAfter time.Duration) error {
	if retryAfter > 0 {
		c.Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
	}
	return apperr.New(apperr.RATE_LIMITED, "Too many requests")
}

// UnauthenticatedRateLimit enforces IP-based rate limits before authentication.
// Runs before RequireAuth so floods of invalid cookies can't hammer the DB.
func (rl *RateLimitMiddleware) UnauthenticatedRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		allowed, err := rl.limiter.AllowIP(c.Context(), "unauthenticated", ip, 20, time.Minute)
		if err != nil {
			slog.Warn("unauthenticated rate limit IP check failed", "err", err)
			return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
		}
		if !allowed {
			return apperr.New(apperr.RATE_LIMITED, "Too many requests")
		}
		return c.Next()
	}
}

// ProtectedRateLimit enforces tenant-scoped rate limits for authenticated endpoints.
// It requires a tenant ID in context (set by auth middleware).
func (rl *RateLimitMiddleware) ProtectedRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID := TenantIDFrom(c.Context())
		ip := c.IP()
		allowed, err := rl.limiter.AllowTenantIP(c.Context(), "api", tenantID, ip, 100, time.Minute)
		if err != nil {
			slog.Warn("protected rate limit tenant IP check failed", "err", err)
			return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
		}
		if !allowed {
			return apperr.New(apperr.RATE_LIMITED, "Too many requests")
		}
		return c.Next()
	}
}

// MutationRateLimit enforces higher tenant-scoped limits for mutations (write operations).
func (rl *RateLimitMiddleware) MutationRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID := TenantIDFrom(c.Context())
		ip := c.IP()
		allowed, err := rl.limiter.AllowTenantIP(c.Context(), "mutation", tenantID, ip, 200, time.Minute)
		if err != nil {
			slog.Warn("mutation rate limit tenant IP check failed", "err", err)
			return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
		}
		if !allowed {
			return apperr.New(apperr.RATE_LIMITED, "Too many requests")
		}
		return c.Next()
	}
}

// TokenExchangeRateLimit enforces IP-based rate limits for token exchange endpoints.
func (rl *RateLimitMiddleware) TokenExchangeRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		allowed, err := rl.limiter.AllowIP(c.Context(), "token-exchange", ip, 10, time.Minute)
		if err != nil {
			slog.Warn("token exchange rate limit IP check failed", "err", err)
			return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
		}
		if !allowed {
			return apperr.New(apperr.RATE_LIMITED, "Too many requests")
		}
		return c.Next()
	}
}

// HealthzRateLimit applies a permissive rate limit for health endpoints.
func (rl *RateLimitMiddleware) HealthzRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		allowed, err := rl.limiter.AllowIP(c.Context(), "healthz", ip, 1000, time.Minute)
		if err != nil {
			slog.Warn("healthz rate limit IP check failed", "err", err)
			return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
		}
		if !allowed {
			return apperr.New(apperr.RATE_LIMITED, "Too many requests")
		}
		return c.Next()
	}
}

// Context helpers
func (rl *RateLimitMiddleware) AllowIP(ctx context.Context, prefix, ip string, limit int, window time.Duration) error {
	allowed, err := rl.limiter.AllowIP(ctx, prefix, ip, limit, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.New(apperr.RATE_LIMITED, "Too many requests")
	}
	return nil
}

func (rl *RateLimitMiddleware) AllowEmail(ctx context.Context, prefix, email string, limit int, window time.Duration) error {
	allowed, err := rl.limiter.AllowEmail(ctx, prefix, email, limit, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.New(apperr.RATE_LIMITED, "Too many requests")
	}
	return nil
}

func (rl *RateLimitMiddleware) AllowTenantIP(ctx context.Context, tenantID, ip string, limit int, window time.Duration) error {
	allowed, err := rl.limiter.AllowTenantIP(ctx, "ip", tenantID, ip, limit, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.New(apperr.RATE_LIMITED, "Too many requests")
	}
	return nil
}

func (rl *RateLimitMiddleware) AllowTenantEmail(ctx context.Context, tenantID, email string, limit int, window time.Duration) error {
	allowed, err := rl.limiter.AllowTenantEmail(ctx, "email", tenantID, email, limit, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.New(apperr.RATE_LIMITED, "Too many requests")
	}
	return nil
}

func (rl *RateLimitMiddleware) AllowGlobal(ctx context.Context, name string, limit int, window time.Duration) error {
	allowed, err := rl.limiter.AllowGlobal(ctx, name, limit, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.New(apperr.RATE_LIMITED, "Too many requests")
	}
	return nil
}

func (rl *RateLimitMiddleware) AllowTenantGlobal(ctx context.Context, tenantID, name string, limit int, window time.Duration) error {
	allowed, err := rl.limiter.AllowTenantGlobal(ctx, tenantID, name, limit, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.New(apperr.RATE_LIMITED, "Too many requests")
	}
	return nil
}
