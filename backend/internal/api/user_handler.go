package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

// userHandler responds to /api/users/*.
type userHandler struct {
	svc services.UserService
}

// newUserHandler wires the dependency.
func newUserHandler(svc services.UserService) *userHandler {
	return &userHandler{svc: svc}
}

func (h *userHandler) getByID(c fiber.Ctx) error {
	id := c.Params("id")
	tenantID := c.Get("X-Tenant-ID")

	user, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return mapServiceError(c, err)
	}
	return c.JSON(user)
}

func (h *userHandler) getByEmail(c fiber.Ctx) error {
	email := c.Params("email")
	tenantID := c.Get("X-Tenant-ID")

	user, err := h.svc.GetByEmail(c.Context(), tenantID, email)
	if err != nil {
		return mapServiceError(c, err)
	}
	return c.JSON(user)
}

// mapServiceError converts service-layer errors into HTTP responses.
// It is the sole place where service errors become HTTP responses.
func mapServiceError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidUUID):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_uuid",
			"message": "invalid user id",
			"errors":  fiber.Map{"id": []string{"must be a valid UUID"}},
		})
	case errors.Is(err, services.ErrTenantRequired):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_tenant",
			"message": "valid X-Tenant-ID required for email lookup",
			"errors":  fiber.Map{"tenant_id": []string{"required UUID"}},
		})
	case errors.Is(err, services.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "not_found",
			"message": "user not found",
			"errors":  fiber.Map{},
		})
	default:
		// Unhandled service error — log and surface a generic 500.
		// Do not leak internal error details to the client.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "An unexpected error occurred",
			"errors":  fiber.Map{},
		})
	}
}
