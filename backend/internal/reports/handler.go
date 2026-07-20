package reports

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes report HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new reports Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts report routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	reports := router.Group("/api/v1/reports")
	reports.Get("/student/:student_id/term/:term_id", middleware.RequireAuth, h.GetTermReport)
}

// repMiddleware extracts common tenant/school context.
func (h *Handler) repMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
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

// GetTermReport handles GET /api/v1/reports/student/:student_id/term/:term_id.
func (h *Handler) GetTermReport(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.repMiddleware(c)
	if err != nil {
		return err
	}

	studentID := c.Params("student_id")
	termID := c.Params("term_id")

	report, err := h.svc.GetTermReport(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(TermReportResponse{Data: *report})
}
