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
// @Description Returns active user counts per role for a school, with student gender breakdown.
// @Tags Schools
// @Produce json
// @Param schoolId path string true "School ID"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Router /schools/{schoolId}/users/count [get]
func (h *UserCountsHandler) GetUserCounts(c fiber.Ctx) error {
	schoolID := c.Params("schoolId")
	if schoolID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: school_id is required")
	}
	counts, err := h.service.GetUserCounts(c.Context(), schoolID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusOK).JSON(counts)
}
