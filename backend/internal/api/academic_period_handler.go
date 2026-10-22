package api

import (
	"somotracker/backend/internal/services"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
)

type AcademicPeriodHandler struct {
	service services.AcademicPeriodService
}

func NewAcademicPeriodHandler(svc services.AcademicPeriodService) *AcademicPeriodHandler {
	return &AcademicPeriodHandler{service: svc}
}

// CreateAcademicPeriod creates a new academic year with nested terms.
//
// @Summary Create academic period
// @Description Creates a new academic year with nested terms for the active school.
// @Tags Academic Periods
// @Accept json
// @Produce json
// @Param body body services.AcademicPeriodRequest true "Academic period payload"
// @Success 201 {object} object
// @Failure 400 {object} object
// @Failure 401 {object} object
// @Router /api/school/academic-period [post]
func (h *AcademicPeriodHandler) CreateAcademicPeriod(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return WriteError(c, ErrUnauthorized("active_school_id not found in session"))
	}

	var body services.AcademicPeriodRequest
	if err := c.Bind().Body(&body); err != nil {
		return WriteError(c, ErrBadRequest("invalid JSON body", nil))
	}

	yearID, err := h.service.CreateAcademicPeriod(c.Context(), schoolID, body)
	if err != nil {
		if se, ok := err.(*services.ServiceError); ok {
			switch se.Code {
			case "bad_request":
				return WriteError(c, ErrBadRequest(se.Message, nil))
			case "forbidden":
				return WriteError(c, ErrForbidden(se.Code, se.Message))
			default:
				go_uber_zap.L().Error("academic_period: create failed", go_uber_zap.Error(err))
				return WriteError(c, ErrInternal("failed to create academic period"))
			}
		}
		go_uber_zap.L().Error("academic_period: create failed", go_uber_zap.Error(err))
		return WriteError(c, ErrInternal("failed to create academic period"))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":             "academic_period_created",
		"message":          "Academic period created successfully",
		"academic_year_id": yearID,
		"errors":           fiber.Map{},
	})
}

// ListAcademicPeriods returns all academic periods for the active school.
//
// @Summary List academic periods
// @Description Returns all academic years with their terms for the active school.
// @Tags Academic Periods
// @Produce json
// @Success 200 {object} object
// @Failure 401 {object} object
// @Router /api/school/academic-period [get]
func (h *AcademicPeriodHandler) ListAcademicPeriods(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return WriteError(c, ErrUnauthorized("active_school_id not found in session"))
	}

	periods, err := h.service.ListAcademicPeriods(c.Context(), schoolID)
	if err != nil {
		if se, ok := err.(*services.ServiceError); ok {
			switch se.Code {
			case "bad_request":
				return WriteError(c, ErrBadRequest(se.Message, nil))
			case "forbidden":
				return WriteError(c, ErrForbidden(se.Code, se.Message))
			default:
				go_uber_zap.L().Error("academic_period: list failed", go_uber_zap.Error(err))
				return WriteError(c, ErrInternal("failed to list academic periods"))
			}
		}
		go_uber_zap.L().Error("academic_period: list failed", go_uber_zap.Error(err))
		return WriteError(c, ErrInternal("failed to list academic periods"))
	}

	return c.JSON(fiber.Map{
		"code":    "academic_periods_listed",
		"message": "Academic periods retrieved successfully",
		"data":    periods,
		"errors":  fiber.Map{},
	})
}
