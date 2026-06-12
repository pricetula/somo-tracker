package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// ============================================================================
// Handler — HTTP delivery layer (requirement 4, 6, 7, 14).
// ============================================================================

const (
	somoCookieName = "somo_sid"
	cookieMaxAge   = 2592000 // 30 days in seconds
)

// ErrorBody is the JSON response body for error responses (requirement 14).
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Handler exposes auth HTTP endpoints.
type Handler struct {
	svc    *Service
	logger *zap.Logger
	cfg    config.Config
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, logger *zap.Logger, cfg config.Config) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
		cfg:    cfg,
	}
}

// RegisterRoutes mounts auth routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/api/auth")

	auth.Post("/discover", h.Discover)
	auth.Post("/verify", h.Verify)
	auth.Post("/register", h.Register)
	auth.Get("/me", h.Me)
	auth.Delete("/session", h.Logout)
}

// Discover handles POST /api/auth/discover (PHASE 1).
func (h *Handler) Discover(c *fiber.Ctx) error {
	var payload DiscoveryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}
	if payload.Email == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "email is required",
		})
	}

	if err := h.svc.Discover(c.Context(), payload.Email); err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	return c.SendStatus(fiber.StatusOK)
}

// Verify handles POST /api/auth/verify (PHASE 2).
func (h *Handler) Verify(c *fiber.Ctx) error {
	var payload struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}
	if payload.Token == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "token is required",
		})
	}

	sessionRef, err := h.svc.Verify(c.Context(), payload.Token)
	if err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	return c.JSON(fiber.Map{
		"session_ref": sessionRef,
	})
}

// Register handles POST /api/auth/register (PHASE 3).
func (h *Handler) Register(c *fiber.Ctx) error {
	var payload RegistrationPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	// Extract device fingerprint from security pipeline (requirement 5)
	deviceFingerprint, _ := c.Locals("device_fingerprint").(string)

	sessionToken, err := h.svc.Register(c.Context(), payload.SessionRef, payload, deviceFingerprint)
	if err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	// Issue HTTPOnly cookie (requirement 4)
	c.Cookie(&fiber.Cookie{
		Name:     somoCookieName,
		Value:    sessionToken,
		HTTPOnly: true,
		Secure:   h.cfg.AppEnv != "development", // Secure in non-dev environments
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   cookieMaxAge,
	})

	return c.SendStatus(fiber.StatusNoContent)
}

// Me handles GET /api/auth/me (requirement 6).
func (h *Handler) Me(c *fiber.Ctx) error {
	token := c.Cookies(somoCookieName)
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "expired_token",
			Message: "no session cookie found",
		})
	}

	session, err := h.svc.GetSession(c.Context(), token)
	if err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	return c.JSON(fiber.Map{
		"user_id":   session.UserID,
		"tenant_id": session.TenantID,
	})
}

// Logout handles DELETE /api/auth/session (requirement 7).
func (h *Handler) Logout(c *fiber.Ctx) error {
	token := c.Cookies(somoCookieName)

	if err := h.svc.Logout(c.Context(), token); err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	// Clear the cookie
	c.Cookie(&fiber.Cookie{
		Name:     somoCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   h.cfg.AppEnv != "development",
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1, // Expire immediately
	})

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError maps domain errors to HTTP status codes and JSON bodies (requirement 14).
func (h *Handler) mapError(err error) (int, ErrorBody) {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return fiber.StatusUnprocessableEntity, ErrorBody{
			Error:   "invalid_input",
			Message: validationErr.Message,
		}
	}

	switch {
	case errors.Is(err, ErrInvalidInput):
		return fiber.StatusUnprocessableEntity, ErrorBody{
			Error:   "invalid_input",
			Message: err.Error(),
		}
	case errors.Is(err, ErrExpiredToken):
		return fiber.StatusUnauthorized, ErrorBody{
			Error:   "expired_token",
			Message: "session expired or invalid",
		}
	case errors.Is(err, ErrMFARequired):
		return fiber.StatusUnauthorized, ErrorBody{
			Error:   "mfa_required",
			Message: "multi-factor authentication challenge required",
		}
	case errors.Is(err, ErrOrgAlreadyExists):
		return fiber.StatusConflict, ErrorBody{
			Error:   "org_already_exists",
			Message: "organization already exists",
		}
	case errors.Is(err, ErrNotFound):
		return fiber.StatusNotFound, ErrorBody{
			Error:   "not_found",
			Message: "resource not found",
		}
	default:
		return fiber.StatusInternalServerError, ErrorBody{
			Error:   "internal_error",
			Message: "an unexpected error occurred",
		}
	}
}
