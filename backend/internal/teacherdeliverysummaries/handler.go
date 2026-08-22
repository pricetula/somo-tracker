package teacherdeliverysummaries

import (
	"context"
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// AcademicYearTermResolver defines the interface for resolving current academic year and term.
type AcademicYearTermResolver interface {
	GetCurrentYearAndTermID(ctx context.Context, tenantID, schoolID string) (yearID, termID string, err error)
}

// Handler exposes teacher delivery summary HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc AcademicYearTermResolver
}

// NewHandler creates a new teacher delivery summaries Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc AcademicYearTermResolver) {
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

// RegisterRoutes mounts teacher delivery summary routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	tds := router.Group("/api/v1/teacher-delivery-summaries")
	tds.Post("/refresh", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.Refresh)
	tds.Get("/", middleware.RequireAuth, h.ListByTerm)
	tds.Get("/teacher/:user_id", middleware.RequireAuth, h.ListByTeacher)
	tds.Get("/breakdown", middleware.RequireAuth, h.ListDeliveryBreakdown)
	tds.Get("/:user_id", middleware.RequireAuth, h.GetSummary)
}

// tdsMiddleware extracts common tenant/school context.
func (h *Handler) tdsMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
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

// Refresh handles POST /api/v1/teacher-delivery-summaries/refresh.
// Academic term is resolved server-side from the current active term if not provided.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tdsMiddleware(c)
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
		Message: "Teacher delivery summaries refreshed",
		TermID:  termID,
	})
}

// ListByTerm handles GET /api/v1/teacher-delivery-summaries.
// Query params: term_id (optional, defaults to current active term).
func (h *Handler) ListByTerm(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tdsMiddleware(c)
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

	result, err := h.svc.ListByTerm(c.Context(), tenantID, schoolID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListByTeacher handles GET /api/v1/teacher-delivery-summaries/teacher/:user_id.
// Query params: term_id (optional, defaults to current active term).
func (h *Handler) ListByTeacher(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tdsMiddleware(c)
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

	result, err := h.svc.ListByTeacher(c.Context(), tenantID, schoolID, userID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListDeliveryBreakdown handles GET /api/v1/teacher-delivery-summaries/breakdown.
//
// Query params:
//   - academic_term_id (UUID, optional) — the term to aggregate
//     (teacher_delivery_summaries are per teacher × term).
//     If not provided, the current active term is used.
//
// tenant_id and school_id are resolved from the authenticated local context.
// Returns per-teacher Marked vs. Missed slot counts ordered by missed_slots
// descending so chronic non-compliant teachers surface first.
func (h *Handler) ListDeliveryBreakdown(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tdsMiddleware(c)
	if err != nil {
		return err
	}

	termID := c.Query("academic_term_id")
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

	result, err := h.svc.ListDeliveryBreakdown(c.Context(), tenantID, schoolID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetSummary handles GET /api/v1/teacher-delivery-summaries/:user_id.
// Query params: term_id (optional, defaults to current active term).
func (h *Handler) GetSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tdsMiddleware(c)
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

	summary, err := h.svc.GetByTeacherTerm(c.Context(), userID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(summary)
}
