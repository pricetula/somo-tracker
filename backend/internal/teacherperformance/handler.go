package teacherperformance

import (
	"github.com/gofiber/fiber/v2"
	"somotracker/backend/internal/academicyears"

	"somotracker/backend/internal/middleware"
)

// Handler exposes teacher performance HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc academicyears.AcademicYearTermResolver
}

// NewHandler creates a new teacher performance Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicyears.AcademicYearTermResolver) {
	h.academicYearsSvc = aySvc
}

// resolveCurrentTerm resolves the current academic term ID for the school.
// Returns empty string if no current term is set.
func (h *Handler) resolveCurrentTerm(c *fiber.Ctx, tenantID, schoolID string) (string, error) {
	if h.academicYearsSvc == nil {
		return "", nil
	}
	_, termID, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	return termID, err
}

// RegisterRoutes mounts teacher performance routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	tp := router.Group("/api/v1/teacher-performance")
	tp.Post("/refresh", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.Refresh)
	tp.Get("/summaries", middleware.RequireAuth, h.ListSummaries)
	tp.Get("/summaries/teacher/:user_id", middleware.RequireAuth, h.ListByTeacher)
	tp.Get("/summaries/:user_id/:learning_area_id/:class_id", middleware.RequireAuth, h.GetSummary)
}

// tpMiddleware extracts common tenant/school from context.
func (h *Handler) tpMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
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

// Refresh handles POST /api/v1/teacher-performance/refresh.
// Academic term is resolved server-side from the current active term if not provided.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tpMiddleware(c)
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

	// Resolve term if not provided
	termID := payload.AcademicTermID
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	if err := h.svc.RefreshComputation(c.Context(), termID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(RefreshResponse{
		Message: "Teacher performance summaries refreshed",
		TermID:  termID,
	})
}

// ListSummaries handles GET /api/v1/teacher-performance/summaries.
// Query params: term_id (optional, defaults to current active term), class_id (optional), learning_area_id (optional).
func (h *Handler) ListSummaries(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tpMiddleware(c)
	if err != nil {
		return err
	}

	termID := c.Query("term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	var classID, learningAreaID *string
	if cid := c.Query("class_id"); cid != "" {
		classID = &cid
	}
	if lid := c.Query("learning_area_id"); lid != "" {
		learningAreaID = &lid
	}

	result, err := h.svc.ListByTerm(c.Context(), tenantID, schoolID, termID, classID, learningAreaID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListByTeacher handles GET /api/v1/teacher-performance/summaries/teacher/:user_id.
// Query params: term_id (optional, defaults to current active term), learning_area_id (optional).
func (h *Handler) ListByTeacher(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tpMiddleware(c)
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

	termID := c.Query("term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	var learningAreaID *string
	if lid := c.Query("learning_area_id"); lid != "" {
		learningAreaID = &lid
	}

	result, err := h.svc.ListByTeacher(c.Context(), tenantID, schoolID, userID, termID, learningAreaID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetSummary handles GET /api/v1/teacher-performance/summaries/:user_id/:learning_area_id/:class_id.
// Query params: term_id (optional, defaults to current active term).
func (h *Handler) GetSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tpMiddleware(c)
	if err != nil {
		return err
	}

	userID := c.Params("user_id")
	learningAreaID := c.Params("learning_area_id")
	classID := c.Params("class_id")

	if userID == "" || learningAreaID == "" || classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "user_id, learning_area_id, and class_id are required",
		})
	}

	termID := c.Query("term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	summary, err := h.svc.GetByTeacherClassSubject(c.Context(), userID, learningAreaID, classID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(summary)
}
