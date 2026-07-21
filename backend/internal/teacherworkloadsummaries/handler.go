package teacherworkloadsummaries

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes teacher workload summary HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new teacher workload summaries Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts teacher workload summary routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	tws := router.Group("/api/v1/teacher-workload-summaries")
	tws.Post("/refresh", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.Refresh)
	tws.Get("/", middleware.RequireAuth, h.ListByYear)
	tws.Get("/teacher/:user_id", middleware.RequireAuth, h.ListByTeacher)
	tws.Get("/:user_id", middleware.RequireAuth, h.GetSummary)
}

// twsMiddleware extracts common tenant/school context.
func (h *Handler) twsMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
	tenantID = c.Locals("tenant_id").(string)
	schoolID, _ = c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	return tenantID, schoolID, nil
}

// Refresh handles POST /api/v1/teacher-workload-summaries/refresh.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	_, _, err := h.twsMiddleware(c)
	if err != nil {
		return err
	}

	var payload RefreshRequest
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.AcademicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id is required",
		})
	}

	if err := h.svc.RefreshComputation(c.Context(), payload.AcademicYearID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(RefreshResponse{
		Message: "Teacher workload summaries refreshed",
		YearID:  payload.AcademicYearID,
	})
}

// ListByYear handles GET /api/v1/teacher-workload-summaries.
// Query params: academic_year_id (required).
func (h *Handler) ListByYear(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.twsMiddleware(c)
	if err != nil {
		return err
	}

	yearID := c.Query("academic_year_id")
	if yearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id is required",
		})
	}

	result, err := h.svc.ListByYear(c.Context(), tenantID, schoolID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListByTeacher handles GET /api/v1/teacher-workload-summaries/teacher/:user_id.
// Query params: academic_year_id (required).
func (h *Handler) ListByTeacher(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.twsMiddleware(c)
	if err != nil {
		return err
	}

	userID := c.Params("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "user_id is required",
		})
	}

	yearID := c.Query("academic_year_id")
	if yearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id is required",
		})
	}

	result, err := h.svc.ListByTeacher(c.Context(), tenantID, schoolID, userID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetSummary handles GET /api/v1/teacher-workload-summaries/:user_id.
// Query params: academic_year_id (required).
func (h *Handler) GetSummary(c *fiber.Ctx) error {
	_, _, err := h.twsMiddleware(c)
	if err != nil {
		return err
	}

	userID := c.Params("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "user_id is required",
		})
	}

	yearID := c.Query("academic_year_id")
	if yearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id is required",
		})
	}

	summary, err := h.svc.GetByTeacherYear(c.Context(), userID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(summary)
}
