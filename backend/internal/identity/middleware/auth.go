package middleware

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/config"
)

// AuthMiddleware validates session token on protected routes
// Per ADR-0002: Go is sole security authority, validates every protected request
type AuthMiddleware struct {
	svc *service.Service
	cfg *config.Config
}

// NewAuthMiddleware creates auth middleware.
func NewAuthMiddleware(svc *service.Service, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{svc: svc, cfg: cfg}
}

// sessionCookieName returns the cookie name with environment suffix per ADR-0002
func (m *AuthMiddleware) sessionCookieName() string {
	base := "__Secure-session_token"
	if m.cfg.Env != "" && m.cfg.Env != "production" {
		return base + "_" + m.cfg.Env
	}
	return base
}

// RequireAuth middleware validates session token from cookie
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Get session token from cookie
		cookieName := m.sessionCookieName()
		token := c.Cookies(cookieName)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "authentication required",
			})
		}

		// Hash token to lookup in database
		tokenHash := sha256.Sum256([]byte(token))

		// Validate session
		sessionCtx, err := m.svc.ValidateSession(c.Context(), tokenHash[:])
		if err != nil {
			if errors.Is(err, service.ErrSessionNotFound) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"code":    "UNAUTHENTICATED",
					"message": "session not found or expired",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "session validation failed",
			})
		}

		// Store session context in locals for handlers
		c.Locals("session_id", sessionCtx.SessionID)
		c.Locals("user_id", sessionCtx.UserID)
		c.Locals("tenant_id", sessionCtx.TenantID)
		c.Locals("role", sessionCtx.Role)

		// Also store in context for service layer
		//nolint:staticcheck
		ctx := context.WithValue(c.Context(), "session_ctx", sessionCtx)
		c.SetContext(ctx)

		return c.Next()
	}
}

// OptionalAuth middleware validates session if present, but doesn't require it
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		cookieName := m.sessionCookieName()
		token := c.Cookies(cookieName)
		if token == "" {
			return c.Next()
		}

		tokenHash := sha256.Sum256([]byte(token))
		sessionCtx, err := m.svc.ValidateSession(c.Context(), tokenHash[:])
		if err != nil {
			// Invalid session - continue without auth
			return c.Next()
		}

		c.Locals("session_id", sessionCtx.SessionID)
		c.Locals("user_id", sessionCtx.UserID)
		c.Locals("tenant_id", sessionCtx.TenantID)
		c.Locals("role", sessionCtx.Role)

		//nolint:staticcheck
		ctx := context.WithValue(c.Context(), "session_ctx", sessionCtx)
		c.SetContext(ctx)

		return c.Next()
	}
}

// CSRFMiddleware validates Origin header for mutations
func (m *AuthMiddleware) CSRFMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Only validate on mutating methods
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		origin := c.Get("Origin")
		secFetchSite := c.Get("Sec-Fetch-Site")

		// Allow if Origin is missing but Sec-Fetch-Site is same-origin (per spec)
		if origin == "" {
			if secFetchSite == "same-origin" {
				return c.Next()
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    "CSRF_FAILED",
				"message": "missing origin header",
			})
		}

		// Validate origin against allowed origins
		// In production, check against configured allowed origins
		allowedOrigins := []string{
			"https://app.example.com",
			"http://localhost:3000",
		}

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    "CSRF_FAILED",
				"message": "invalid origin",
			})
		}

		return c.Next()
	}
}

// TenantMiddleware extracts tenant from session and validates access
func (m *AuthMiddleware) TenantMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    "FORBIDDEN",
				"message": "tenant not found in session",
			})
		}

		// Store tenant ID in context for repository queries
		//nolint:staticcheck
		ctx := context.WithValue(c.Context(), "tenant_id", tenantID)
		c.SetContext(ctx)

		return c.Next()
	}
}
