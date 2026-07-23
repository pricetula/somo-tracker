package cohortpositions

import (
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for cohort position summaries.
type Handler struct {
	svc *Service
}

// NewHandler creates a new cohort positions HTTP handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers cohort position routes on the Fiber app.
// Routes:
//
//	POST /api/v1/cohort-positions/refresh  — trigger a batch refresh for a term
//	GET  /api/v1/cohort-positions/:studentId — get a student's position
//	GET  /api/v1/cohort-positions/class/:classId — list class positions
//	GET  /api/v1/cohort-positions/grade/:gradeLevel — list grade positions
func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1/cohort-positions")

	api.Post("/refresh", h.RefreshTerm)
	api.Get("/:studentId", h.GetByStudent)
	api.Get("/class/:classId", h.ListByClass)
	api.Get("/grade/:gradeLevel", h.ListByGrade)
}

// RefreshTerm triggers a batch refresh of cohort positions for a given term.
// POST /api/v1/cohort-positions/refresh
// Body: { "academic_term_id": "uuid" }
func (h *Handler) RefreshTerm(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if req.AcademicTermID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "academic_term_id is required",
		})
	}

	if err := h.svc.RefreshTerm(c.Context(), req.AcademicTermID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "refresh_failed",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(RefreshResponse{
		Message: "cohort positions refresh initiated",
		TermID:  req.AcademicTermID,
	})
}

// GetByStudent returns the cohort position for a student in a term.
// GET /api/v1/cohort-positions/:studentId?term_id=xxx
func (h *Handler) GetByStudent(c *fiber.Ctx) error {
	studentID := c.Params("studentId")
	termID := c.Query("term_id")

	// Resolve tenant_id from the request context (set by auth middleware).
	tenantID := c.Locals("tenant_id").(string)

	position, err := h.svc.GetByStudentTerm(c.Context(), studentID, termID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "not_found",
			"message": "cohort position not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": position,
	})
}

// ListByClass returns all cohort positions for a class in a term.
// GET /api/v1/cohort-positions/class/:classId?term_id=xxx
func (h *Handler) ListByClass(c *fiber.Ctx) error {
	classID := c.Params("classId")
	termID := c.Query("term_id")
	tenantID := c.Locals("tenant_id").(string)

	positions, err := h.svc.ListByClassTerm(c.Context(), classID, termID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list class positions",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"items": positions,
		"total": len(positions),
	})
}

// ListByGrade returns all cohort positions at a grade level in a term.
// GET /api/v1/cohort-positions/grade/:gradeLevel?school_id=xxx&term_id=xxx
func (h *Handler) ListByGrade(c *fiber.Ctx) error {
	gradeLevel := c.Params("gradeLevel")
	termID := c.Query("term_id")
	schoolID := c.Query("school_id")
	tenantID := c.Locals("tenant_id").(string)

	positions, err := h.svc.ListByGradeTerm(c.Context(), schoolID, gradeLevel, termID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list grade positions",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"items": positions,
		"total": len(positions),
	})
}
