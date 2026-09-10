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

// getMe returns the current authenticated user's profile.
//
// @Summary Get current user
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/me [get]
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
	case strings.Contains(msg, "bad_request:"):
		parts := strings.SplitN(msg, "bad_request: ", 2)
		message := msg
		if len(parts) == 2 {
			message = parts[1]
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": message,
			"errors":  fiber.Map{},
		})
	case strings.Contains(msg, "not_found:"):
		parts := strings.SplitN(msg, "not_found: ", 2)
		message := msg
		if len(parts) == 2 {
			message = parts[1]
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "not_found",
			"message": message,
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
