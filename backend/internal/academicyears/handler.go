package academicyears

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler
// ============================================================================

// Handler exposes academic year and term HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts academic calendar routes on the given router.
//
// Years are read-only via the API (year creation/activation is driven by the
// term lifecycle and SetupInitialYear during school registration). All term
// mutations are admin-only.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Academic Years (read-only)
	years := router.Group("/api/v1/academic-years")
	years.Get("/", middleware.RequireAuth, h.ListYears)
	years.Get("/current", middleware.RequireAuth, h.GetCurrent)

	// Academic Terms
	terms := router.Group("/api/v1/academic-terms")
	terms.Get("/", middleware.RequireAuth, h.ListTerms)
	terms.Post("/", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.CreateTerm)
	terms.Patch("/:id", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.PatchTerm)
	terms.Post("/:id/activate", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.ActivateTerm)
	terms.Delete("/:id", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.DeleteTerm)
}

// ============================================================================
// Error response helper — matches the canonical { code, message, details } shape
// ============================================================================

type errorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string, details interface{}) error {
	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// ============================================================================
// YEARS
// ============================================================================

// GetCurrent handles GET /api/v1/academic-years/current.
func (h *Handler) GetCurrent(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)

	// School scope is implicit from the session — use tenant scope
	// In production, derive the active school from member_active_school

	years, err := h.svc.GetCurrent(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(years)
}

// ListYears handles GET /api/v1/academic-years.
func (h *Handler) ListYears(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)

	// school_id comes from the active school context
	// School scope is implicit from the session — use tenant scope
	// In production, derive the active school from member_active_school

	years, err := h.svc.ListYears(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"data": years,
	})
}

// ============================================================================
// TERMS
// ============================================================================

// ListTerms handles GET /api/v1/academic-terms.
func (h *Handler) ListTerms(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)

	var academicYearID *string
	if ayID := c.Query("academic_year_id"); ayID != "" {
		academicYearID = &ayID
	}

	terms, err := h.svc.ListTerms(c.Context(), tenantID, schoolID, academicYearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"data": terms,
	})
}

// CreateTerm handles POST /api/v1/academic-terms.
func (h *Handler) CreateTerm(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	schoolID := c.Locals("school_id").(string)

	var body CreateTermBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	term, err := h.svc.CreateTerm(c.Context(), body, tenantID, schoolID, userID, nil)
	if err != nil {
		var outOfBounds *TermOutOfYearBoundsError
		if errors.As(err, &outOfBounds) {
			return writeError(c, fiber.StatusUnprocessableEntity, "TERM_OUT_OF_YEAR_BOUNDS", outOfBounds.Error(), nil)
		}
		var overlap *TermDateOverlapError
		if errors.As(err, &overlap) {
			return writeError(c, fiber.StatusUnprocessableEntity, "TERM_DATE_OVERLAP",
				overlap.Error(), fiber.Map{
					"conflicting_term": overlap.ConflictingName,
				})
		}
		var numExists *TermNumberExistsError
		if errors.As(err, &numExists) {
			return writeError(c, fiber.StatusConflict, "TERM_NUMBER_EXISTS", numExists.Error(), nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(term)
}

// PatchTerm handles PATCH /api/v1/academic-terms/:id.
func (h *Handler) PatchTerm(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	id := c.Params("id")
	schoolID := c.Locals("school_id").(string)

	var body PatchTermBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.Version == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "version is required for optimistic locking", nil)
	}

	// Strip is_current if present
	warnings := []string{}
	var raw map[string]interface{}
	if err := c.BodyParser(&raw); err == nil {
		if _, exists := raw["is_current"]; exists {
			warnings = append(warnings, "is_current cannot be set via PATCH. It is managed automatically.")
		}
		if _, exists := raw["term_number"]; exists {
			warnings = append(warnings, "term_number cannot be changed via PATCH.")
		}
	}

	term, err := h.svc.PatchTerm(c.Context(), id, tenantID, schoolID, body, userID, nil)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return writeError(c, fiber.StatusConflict, "conflict",
				"Resource was modified by another request. Fetch the latest version and retry.", nil)
		}
		var outOfBounds *TermOutOfYearBoundsError
		if errors.As(err, &outOfBounds) {
			return writeError(c, fiber.StatusBadRequest, "TERM_OUT_OF_YEAR_BOUNDS", outOfBounds.Error(), nil)
		}
		var overlap *TermDateOverlapError
		if errors.As(err, &overlap) {
			return writeError(c, fiber.StatusConflict, "TERM_DATE_OVERLAP",
				overlap.Error(), fiber.Map{
					"conflicting_term": overlap.ConflictingName,
				})
		}
		var orphaned *OrphanedRecordsError
		if errors.As(err, &orphaned) {
			return writeError(c, fiber.StatusConflict, "ORPHANED_RECORDS", orphaned.Error(), fiber.Map{
				"assessment_sessions": orphaned.Assessments,
				"attendance_records":  orphaned.AttendanceMarks,
				"start_date":          orphaned.StartDate,
				"end_date":            orphaned.EndDate,
			})
		}
		return middleware.HTTPError(c, err)
	}

	resp := fiber.Map{
		"id":               term.ID,
		"name":             term.Name,
		"term_number":      term.TermNumber,
		"start_date":       term.StartDate.Format("2006-01-02"),
		"end_date":         term.EndDate.Format("2006-01-02"),
		"is_current":       term.IsCurrent,
		"is_final":         term.IsFinal,
		"academic_year_id": term.AcademicYearID,
		"version":          term.Version,
	}

	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}

	return c.JSON(resp)
}

// ActivateTerm handles POST /api/v1/academic-terms/:id/activate.
func (h *Handler) ActivateTerm(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	schoolID := c.Locals("school_id").(string)
	id := c.Params("id")

	term, err := h.svc.ActivateTerm(c.Context(), id, tenantID, schoolID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"id":               term.ID,
		"name":             term.Name,
		"term_number":      term.TermNumber,
		"start_date":       term.StartDate.Format("2006-01-02"),
		"end_date":         term.EndDate.Format("2006-01-02"),
		"is_current":       term.IsCurrent,
		"is_final":         term.IsFinal,
		"academic_year_id": term.AcademicYearID,
		"version":          term.Version,
		"message":          "Academic term activated.",
	})
}

// DeleteTerm handles DELETE /api/v1/academic-terms/:id.
func (h *Handler) DeleteTerm(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	schoolID := c.Locals("school_id").(string)
	id := c.Params("id")
	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "id is required", nil)
	}

	err := h.svc.DeleteTerm(c.Context(), id, tenantID, schoolID, userID, nil)
	if err != nil {
		var hasDeps *HasDependentsError
		if errors.As(err, &hasDeps) {
			return writeError(c, fiber.StatusConflict, "HAS_DEPENDENTS", hasDeps.Message,
				fiber.Map{"counts": hasDeps.Counts})
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
