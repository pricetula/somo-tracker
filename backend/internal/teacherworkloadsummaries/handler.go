package teacherworkloadsummaries

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// academicYearsAdapter is the subset of academicyears.Service that the handler uses.
type academicYearsAdapter interface {
	GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error)
	GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error)
}

// Handler exposes teacher workload summary HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc academicYearsAdapter
}

// NewHandler creates a new teacher workload summaries Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicYearsAdapter) {
	h.academicYearsSvc = aySvc
}

// resolveCurrentYear resolves the current academic year ID for the school.
// Returns empty string if no current year is set.
func (h *Handler) resolveCurrentYear(c *fiber.Ctx, tenantID, schoolID string) (string, error) {
	if h.academicYearsSvc == nil {
		return "", nil
	}
	return h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
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
// Academic year is resolved server-side from the current active year if not provided.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.twsMiddleware(c)
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

	// Resolve year if not provided
	yearID := payload.AcademicYearID
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	if err := h.svc.RefreshComputation(c.Context(), yearID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(RefreshResponse{
		Message: "Teacher workload summaries refreshed",
		YearID:  yearID,
	})
}

// ListByYear handles GET /api/v1/teacher-workload-summaries.
// Query params: academic_year_id (optional, defaults to current active year).
func (h *Handler) ListByYear(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.twsMiddleware(c)
	if err != nil {
		return err
	}

	yearID := c.Query("academic_year_id")
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	result, err := h.svc.ListByYear(c.Context(), tenantID, schoolID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListByTeacher handles GET /api/v1/teacher-workload-summaries/teacher/:user_id.
// Query params: academic_year_id (optional, defaults to current active year).
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
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	result, err := h.svc.ListByTeacher(c.Context(), tenantID, schoolID, userID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetSummary handles GET /api/v1/teacher-workload-summaries/:user_id.
// Query params: academic_year_id (optional, defaults to current active year).
func (h *Handler) GetSummary(c *fiber.Ctx) error {
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
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	summary, err := h.svc.GetByTeacherYear(c.Context(), userID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(summary)
}
