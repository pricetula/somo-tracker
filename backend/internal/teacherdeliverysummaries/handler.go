package teacherdeliverysummaries

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes teacher delivery summary HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new teacher delivery summaries Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
func (h *Handler) Refresh(c *fiber.Ctx) error {
	_, _, err := h.tdsMiddleware(c)
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

	if payload.AcademicTermID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_term_id is required",
		})
	}

	if err := h.svc.RefreshComputation(c.Context(), payload.AcademicTermID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(RefreshResponse{
		Message: "Teacher delivery summaries refreshed",
		TermID:  payload.AcademicTermID,
	})
}

// ListByTerm handles GET /api/v1/teacher-delivery-summaries.
// Query params: term_id (required).
func (h *Handler) ListByTerm(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tdsMiddleware(c)
	if err != nil {
		return err
	}

	termID := c.Query("term_id")
	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "term_id is required",
		})
	}

	result, err := h.svc.ListByTerm(c.Context(), tenantID, schoolID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListByTeacher handles GET /api/v1/teacher-delivery-summaries/teacher/:user_id.
// Query params: term_id (required).
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "term_id is required",
		})
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
//   - academic_term_id (UUID, required) — the term to aggregate
//     (teacher_delivery_summaries are per teacher × term).
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_term_id is required",
		})
	}

	result, err := h.svc.ListDeliveryBreakdown(c.Context(), tenantID, schoolID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetSummary handles GET /api/v1/teacher-delivery-summaries/:user_id.
// Query params: term_id (required).
func (h *Handler) GetSummary(c *fiber.Ctx) error {
	_, _, err := h.tdsMiddleware(c)
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "term_id is required",
		})
	}

	summary, err := h.svc.GetByTeacherTerm(c.Context(), userID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(summary)
}
