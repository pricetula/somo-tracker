package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

// meHandler responds to /api/me.
type meHandler struct {
	svc services.MeService
}

func newMeHandler(svc services.MeService) *meHandler {
	return &meHandler{svc: svc}
}

func (h *meHandler) getMe(c fiber.Ctx) error {
	// Extract the session token from the HTTP-only cookie.
	token := c.Cookies("session_token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "session token missing",
			"errors":  fiber.Map{},
		})
	}

	// The session middleware injects tenant context for RLS.
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "tenant context missing",
			"errors":  fiber.Map{},
		})
	}

	result, err := h.svc.GetCurrentUser(c.Context(), token, tenantID)
	if err != nil {
		return mapMeError(c, err)
	}

	return c.JSON(result)
}

func mapMeError(c fiber.Ctx, err error) error {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "bad_request:"):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": strings.TrimPrefix(msg, "bad_request: "),
			"errors":  fiber.Map{},
		})
	case strings.HasPrefix(msg, "not_found:"):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "not_found",
			"message": strings.TrimPrefix(msg, "not_found: "),
			"errors":  fiber.Map{},
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "An unexpected error occurred",
			"errors":  fiber.Map{},
		})
	}
}
