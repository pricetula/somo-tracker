package api

import (
	"github.com/gofiber/fiber/v3"
	"somotracker/backend/internal/services"
)

type UserCountsHandler struct {
	service *services.UserCountsService
}

func NewUserCountsHandler(svc *services.UserCountsService) *UserCountsHandler {
	return &UserCountsHandler{service: svc}
}

// GetUserCounts returns active user counts for a school.
// @Summary Get user counts
// @Description Returns active user counts per role for the active school from session.
// @Tags Schools
// @Produce json
// @Success 200 {object} object
// @Failure 401 {object} object
// @Router /school/users/count [get]
func (h *UserCountsHandler) GetUserCounts(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}
	counts, err := h.service.GetUserCounts(c.Context(), schoolID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusOK).JSON(counts)
}
