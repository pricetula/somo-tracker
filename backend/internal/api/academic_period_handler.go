package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
	"somotracker/backend/internal/services"
)

type AcademicPeriodHandler struct {
	service services.AcademicPeriodService
}

func NewAcademicPeriodHandler(svc services.AcademicPeriodService) *AcademicPeriodHandler {
	return &AcademicPeriodHandler{service: svc}
}

func (h *AcademicPeriodHandler) CreateAcademicPeriod(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}

	var body services.AcademicPeriodRequest
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}

	yearID, err := h.service.CreateAcademicPeriod(c.Context(), schoolID, body)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("academic_period: create failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to create academic period")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":             "academic_period_created",
		"message":          "Academic period created successfully",
		"academic_year_id": yearID,
		"errors":           fiber.Map{},
	})
}
