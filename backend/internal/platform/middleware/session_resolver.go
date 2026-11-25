// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"crypto/sha256"
	"time"

	"github.com/gofiber/fiber/v3"

	"somotracker/internal/auth/store"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// SessionResolver resolves session tokens to user IDs and attaches them to context.
type SessionResolver struct {
	store      store.StoreInterface
	cfg        *config.Config
	log        *logger.Logger
	cookieName string
}

// NewSessionResolver creates a new session resolver middleware.
func NewSessionResolver(store store.StoreInterface, cfg *config.Config) *SessionResolver {
	return &SessionResolver{
		store:      store,
		cfg:        cfg,
		log:        logger.New("info"),
		cookieName: cfg.SessionCookieName,
	}
}

// Resolve returns middleware that resolves the session token from cookie
// and attaches userID to context if valid.
func (sr *SessionResolver) Resolve() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Try to get session token from cookie
		sessionToken := c.Cookies(sr.cookieName)
		if sessionToken == "" {
			// No session cookie, continue without userID
			return c.Next()
		}

		// Hash the token to look up the session
		tokenHash := HashToken(sessionToken)

		session, err := sr.store.GetSessionByTokenHash(c.Context(), tokenHash)
		if err != nil {
			// Invalid session, continue without userID
			return c.Next()
		}

		// Check if session is expired or revoked
		if session.RevokedAt.Valid || session.ExpiresAt.Before(time.Now()) {
			return c.Next()
		}

		// Attach userID to context for rate limiting and authorization
		c.Locals("userID", session.UserID.String())
		c.Locals("organizationID", session.OrganizationID.String())
		c.Locals("sessionType", session.SessionType)
		if session.ActiveSchoolID.Valid {
			c.Locals("activeSchoolID", session.ActiveSchoolID.UUID.String())
		}

		return c.Next()
	}
}

// RequireSession returns middleware that requires a valid session.
// Returns 401 if no valid session found.
func (sr *SessionResolver) RequireSession() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID := c.Locals("userID")
		if userID == nil {
			return apperr.New(apperr.UNAUTHENTICATED, "session required")
		}
		return c.Next()
	}
}

// GetUserIDFromContext retrieves the userID from context.
func GetUserIDFromContext(c fiber.Ctx) string {
	if id := c.Locals("userID"); id != nil {
		return id.(string)
	}
	return ""
}

// GetOrganizationIDFromContext retrieves the organizationID from context.
func GetOrganizationIDFromContext(c fiber.Ctx) string {
	if id := c.Locals("organizationID"); id != nil {
		return id.(string)
	}
	return ""
}

// GetSessionTypeFromContext retrieves the session type from context.
func GetSessionTypeFromContext(c fiber.Ctx) string {
	if st := c.Locals("sessionType"); st != nil {
		return st.(string)
	}
	return ""
}

// HashToken computes SHA-256 of the raw token.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}