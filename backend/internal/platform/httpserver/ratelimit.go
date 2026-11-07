package httpserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/ratelimit"
)

// RateLimitMiddleware provides rate limiting for Fiber routes based on IP, email, and tenant context.
// It wraps internal/platform/ratelimit and exposes helper methods and Fiber handlers.
type RateLimitMiddleware struct {
	limiter *ratelimit.Limiter
}

// NewRateLimitMiddleware creates a new middleware instance wrapping the limiter.
func NewRateLimitMiddleware(limiter *ratelimit.Limiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{limiter: limiter}
}

// PublicRateLimit enforces public rate limits on unauthenticated endpoints:
//  - IP-based limit for magic-link requests
//  - Email-based limit for magic-link requests (when email is present)
// Best used on public API groups before handler execution.
func (rl *RateLimitMiddleware) PublicRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		// IP limit for magic-link requests
		ip := c.IP()
		allowed, err := rl.limiter.AllowIP(c.Context(), "magic-link", ip, 10, time.Minute)
		if err != nil {
			slog.Warn("public rate limit IP check failed", "err", err)
			return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
		}
		if !allowed {
			return apperr.New(apperr.RATE_LIMITED, "Too many requests")
		}

		// Email limit for magic-link requests (if body contains email)
		if email := c.FormValue("email"); email != "" {
			allowed, err = rl.limiter.AllowEmail(c.Context(), "magic-link", email, 3, 15*time.Minute)
			if err != nil {
				slog.Warn("public rate limit email check failed", "err", err)
				return apperr.New(apperr.SERVICE_UNAVAILABLE, "rate limit check unavailable")
			}
			if !allowed {
				return apperr.New(apperr.RATE_LIMITED, "Too many requests")
			}
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

// Legacy helper for direct binding to limiter.
func ApplyRateLimit(rl *RateLimitMiddleware, handler func(c fiber.Ctx) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := handler(c); err != nil {
			return err
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
