package api

import (
	"github.com/go-redis/redis_rate/v10"
	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/api/middleware/ratelimit"
	"somotracker/backend/internal/services"
)

// authRate defines the per-client limit for magic-link initiation.
var authRate = redis_rate.PerMinute(10)

// Router wires all delivery-layer routes. It depends only on the service
// interfaces (not concrete implementations), which makes it fully testable
// with mock services.
type Router struct {
	User    *userHandler
	Tenant  *tenantHandler
	Auth    *authHandler
	limiter *redis_rate.Limiter
}

// NewRouter creates a Router from the injected services and the Redis
// rate-limiting limiter singleton.
func NewRouter(
	userSvc services.UserService,
	tenantSvc services.TenantService,
	authSvc services.AuthService,
	limiter *redis_rate.Limiter,
) *Router {
	return &Router{
		User:    newUserHandler(userSvc),
		Tenant:  newTenantHandler(tenantSvc),
		Auth:    newAuthHandler(authSvc),
		limiter: limiter,
	}
}

// RegisterRoutes attaches the grouped endpoints to the Fiber app.
func (r *Router) RegisterRoutes(app *fiber.App) {
	// Auth routes — protected by Redis-backed rate limiting.
	// The magic-link initiation endpoint is scoped to "api:auth:magic-link".
	app.Post("/api/auth/magic-link/send",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRate, "api:auth:magic-link"),
		r.Auth.sendMagicLink,
	)

	// Magic-link callback (GET for browser redirects from Stytch).
	// Rate-limited independently from sendMagicLink because it's public-facing.
	app.Get("/api/auth/callback",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRate, "api:auth:callback"),
		r.Auth.callback,
	)
}
