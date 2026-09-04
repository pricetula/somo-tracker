package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

// tenantHandler responds to /api/tenants/*.
type tenantHandler struct {
	svc services.TenantService
}

func newTenantHandler(svc services.TenantService) *tenantHandler {
	return &tenantHandler{svc: svc}
}

func (h *tenantHandler) getBySlug(c fiber.Ctx) error {
	slug := c.Params("slug")

	t, err := h.svc.GetBySlug(c.Context(), slug)
	if err != nil {
		return mapTenantError(c, err)
	}
	return c.JSON(t)
}

func mapTenantError(c fiber.Ctx, err error) error {
	if errors.Is(err, services.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "not_found",
			"message": "tenant not found",
			"errors":  fiber.Map{},
		})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"code":    "internal_error",
		"message": "An unexpected error occurred",
		"errors":  fiber.Map{},
	})
}
