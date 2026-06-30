package attendance

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes attendance HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts attendance routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	att := router.Group("/api/v1/schools/:schoolId/attendance")
	att.Post("/", middleware.RequireAuth, h.SubmitAttendance)
	att.Get("/periods/:periodId", middleware.RequireAuth, h.GetPeriod)
}

// SubmitAttendance handles POST /api/v1/schools/:schoolId/attendance.
func (h *Handler) SubmitAttendance(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	schoolID := c.Params("schoolId")

	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id is required",
		})
	}

	var input MarkAttendanceInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	// Set school ID from the path parameter (never trust it from the body)
	input.SchoolID = schoolID

	if err := h.svc.OpenAndSubmitAttendance(c.Context(), tenantID, userID, input); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "success",
		"message": "attendance recorded successfully",
	})
}

// GetPeriod handles GET /api/v1/schools/:schoolId/attendance/periods/:periodId.
func (h *Handler) GetPeriod(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	periodID := c.Params("periodId")
	schoolID := c.Params("schoolId")

	if schoolID == "" || periodID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id and period_id are required",
		})
	}

	period, logs, err := h.svc.GetPeriod(c.Context(), tenantID, periodID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"period": period,
		"logs":   logs,
	})
}
