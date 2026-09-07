package api

import (
	"github.com/go-redis/redis_rate/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/api/middleware/captcha"
	"somotracker/backend/internal/api/middleware/csrf"
	"somotracker/backend/internal/api/middleware/ipblacklist"
	"somotracker/backend/internal/api/middleware/ratelimit"
	"somotracker/backend/internal/api/middleware/session"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/services"
)

// Rate limit tiers for auth endpoints.
// IP-based: first line of defense against distributed attacks.
// Email-based: stricter limit to prevent email bombing/enumeration.
var (
	authRateIP    = redis_rate.PerMinute(10) // 10 req/min per IP
	authRateEmail = redis_rate.PerHour(3)    // 3 req/hour per email
)

// Router wires all delivery-layer routes. It depends only on the service
// interfaces (not concrete implementations), which makes it fully testable
// with mock services.
type Router struct {
	User    *userHandler
	Tenant  *tenantHandler
	Auth    *authHandler
	Me      *meHandler
	limiter *redis_rate.Limiter
	cfg     *config.Config
}

// NewRouter creates a Router from the injected services and the Redis
// rate-limiting limiter singleton. It also accepts a redis.Client for the
// session middleware that protects authenticated routes.
func NewRouter(
	userSvc services.UserService,
	tenantSvc services.TenantService,
	authSvc services.AuthService,
	meSvc services.MeService,
	limiter *redis_rate.Limiter,
	cfg *config.Config,
) *Router {
	return &Router{
		User:    newUserHandler(userSvc),
		Tenant:  newTenantHandler(tenantSvc),
		Auth:    newAuthHandler(authSvc),
		Me:      newMeHandler(meSvc),
		limiter: limiter,
		cfg:     cfg,
	}
}

// RegisterRoutes attaches the grouped endpoints to the Fiber app.
// Routes are split into public (auth-related) and protected groups.
func (r *Router) RegisterRoutes(app *fiber.App, redisClient *redis.Client, logger *zap.Logger) {
	// IP blacklist middleware - checks blacklist before any other processing.
	// Uses fail-open behavior: Redis errors allow request through.
	app.Use(ipblacklist.NewIPBlacklistMiddleware(redisClient, logger, ipblacklist.DefaultConfig()))

	// CAPTCHA middleware for abuse-prone endpoints.
	// Reads config to determine if enabled and which provider to use.
	captchaMW := captcha.NewCaptchaMiddleware(captcha.Config{
		Enabled:        r.cfg.CAPTCHAEnabled,
		ProviderName:   r.cfg.CAPTCHAProvider,
		SiteKey:        r.cfg.CAPTCHASiteKey,
		SecretKey:      r.cfg.CAPTCHASecretKey,
		ScoreThreshold: r.cfg.CAPTCHAScoreThreshold,
	}, logger)

	// ─── Public auth routes (no session, no CSRF) ──────────────────────
	// These are registered directly on the app with full paths to avoid
	// inheriting middleware from the protected group below.

	// Magic-link send: dual rate limits (IP + email) + optional CAPTCHA
	// IP limit: 10/min — catches distributed botnets
	// Email limit: 3/hour — prevents email bombing/enumeration per address
	// CAPTCHA: enabled via config (disabled by default for local dev)
	app.Post("/api/auth/magic-link/send",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:magic-link:ip"),
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateEmail, "api:auth:magic-link:email"),
		captchaMW,
		r.Auth.sendMagicLink,
	)

	app.Get("/api/auth/callback",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:callback"),
		r.Auth.callback,
	)

	// ─── Protected routes (session + CSRF) ─────────────────────────────
	// All routes under /api except the public auth endpoints above.
	protected := app.Group("/api",
		session.NewSessionMiddleware(redisClient, logger),
		csrf.NewCSRFMiddleware(),
	)

	// Logout - requires session + CSRF (mutating request)
	protected.Post("/auth/logout", r.Auth.logout)

	// Protected resources — session middleware validates session cookie,
	// injects user_id and tenant_id into c.Locals, and binds RLS context.
	// CSRF middleware validates double-submit token on mutating requests.
	protected.Get("/users/:id", r.User.getByID)
	protected.Get("/users/email/:email", r.User.getByEmail)
	protected.Get("/tenants/slug/:slug", r.Tenant.getBySlug)
	protected.Get("/me", r.Me.getMe)
}
