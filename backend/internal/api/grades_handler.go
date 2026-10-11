package api

import (
	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
	"somotracker/backend/internal/services"
)

type GradesHandler struct {
	service services.GradesService
}

func NewGradesHandler(svc services.GradesService) *GradesHandler {
	return &GradesHandler{service: svc}
}

// GetGrades returns grade levels for the active school's education system.
//
// @Summary Get grade levels
// @Tags Grades
// @Produce json
// @Success 200 {array} object
// @Failure 400 {object} object
// @Failure 401 {object} object
// @Router /grades [get]
func (h *GradesHandler) GetGrades(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return WriteError(c, ErrUnauthorized("active_school_id not found in session"))
	}

	grades, err := h.service.GetGradeLevels(c.Context(), schoolID)
	if err != nil {
		if se, ok := err.(*services.ServiceError); ok && se.Code == "bad_request" {
			return WriteError(c, ErrBadRequest(se.Message, nil))
		}
		go_uber_zap.L().Error("grades: get failed", go_uber_zap.Error(err))
		return WriteError(c, ErrInternal("failed to get grades"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "grades_fetched",
		"message": "Grade levels fetched successfully",
		"grades":  grades,
		"errors":  fiber.Map{},
	})
}
