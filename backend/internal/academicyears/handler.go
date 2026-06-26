package academicyears

import (
	"errors"
	"strings"

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
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Academic Years
	years := router.Group("/api/v1/academic-years")
	years.Get("/", h.requireAuth, h.ListYears)
	years.Patch("/:id", h.requireAdmin, h.PatchYear)
	years.Post("/:id/set-current", h.requireAdmin, h.SetCurrentYear)
	years.Delete("/:id", h.requireAdmin, h.DeleteYear)

	// Academic Terms
	terms := router.Group("/api/v1/academic-terms")
	terms.Get("/", h.requireAuth, h.ListTerms)
	terms.Post("/", h.requireAdmin, h.CreateTerm)
	terms.Patch("/:id", h.requireAdmin, h.PatchTerm)
	terms.Delete("/:id", h.requireAdmin, h.DeleteTerm)
}

// ============================================================================
// Auth helpers
// ============================================================================

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	session := middleware.GetSession(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "authentication required",
		})
	}
	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	c.Locals("role", session.Role)
	return c.Next()
}

func (h *Handler) requireAdmin(c *fiber.Ctx) error {
	session := middleware.GetSession(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "authentication required",
		})
	}

	role := strings.ToUpper(session.Role)
	if role != "SCHOOL_ADMIN" && role != "SYSTEM_ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    "forbidden",
			"message": "insufficient permissions",
		})
	}

	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	c.Locals("role", session.Role)
	return c.Next()
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

// ListYears handles GET /api/v1/academic-years.
func (h *Handler) ListYears(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	_ = userID // used for future role-based field filtering

	// school_id comes from the active school context
	schoolID := c.Query("school_id")
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
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

// PatchYear handles PATCH /api/v1/academic-years/:id.
func (h *Handler) PatchYear(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	id := c.Params("id")

	var body PatchYearBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	// Optimistic lock: version is required
	if body.Version == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "version is required for optimistic locking", nil)
	}

	// Strip is_current if present in raw JSON
	var raw map[string]interface{}
	if err := c.BodyParser(&raw); err == nil {
		if _, exists := raw["is_current"]; exists {
			// Silently strip and warn
			c.Locals("warnings", []string{"is_current cannot be set via PATCH. Use POST /api/v1/academic-years/:id/set-current."})
		}
	}

	// school_id — in production, derive from active school membership
	schoolID := c.Locals("school_id").(string)

	year, strandingErr := h.svc.PatchYear(c.Context(), id, tenantID, schoolID, body, userID)
	if strandingErr != nil {
		return writeError(c, fiber.StatusUnprocessableEntity, "TERMS_OUT_OF_RANGE",
			strandingErr.Error(), fiber.Map{
				"conflicting_terms": strandingErr.ConflictingTerms,
			})
	}
	if year == nil {
		// Version mismatch or not found — our PatchYear doesn't distinguish cleanly
		// In production, handle with specific error types
		return writeError(c, fiber.StatusConflict, "conflict",
			"Resource was modified by another request. Fetch the latest version and retry.", nil)
	}

	resp := fiber.Map{
		"id":         year.ID,
		"name":       year.Name,
		"start_date": year.StartDate.Format("2006-01-02"),
		"end_date":   year.EndDate.Format("2006-01-02"),
		"is_current": year.IsCurrent,
		"version":    year.Version,
	}

	if warnings, ok := c.Locals("warnings").([]string); ok && len(warnings) > 0 {
		resp["warnings"] = warnings
	}

	return c.JSON(resp)
}

// SetCurrentYear handles POST /api/v1/academic-years/:id/set-current.
func (h *Handler) SetCurrentYear(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	id := c.Params("id")
	schoolID := c.Locals("school_id").(string)

	if err := h.svc.SetCurrentYear(c.Context(), id, tenantID, schoolID, userID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(SetCurrentResponse{Message: "Academic year set as current."})
}

// DeleteYear handles DELETE /api/v1/academic-years/:id.
func (h *Handler) DeleteYear(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	id := c.Params("id")
	schoolID := c.Locals("school_id").(string)

	err := h.svc.DeleteYear(c.Context(), id, tenantID, schoolID, userID)
	if err != nil {
		var hasDeps *HasDependentsError
		if errors.As(err, &hasDeps) {
			return writeError(c, fiber.StatusConflict, "HAS_DEPENDENTS", hasDeps.Message, nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
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
			return writeError(c, fiber.StatusUnprocessableEntity, "TERM_OUT_OF_YEAR_BOUNDS", outOfBounds.Error(), nil)
		}
		var overlap *TermDateOverlapError
		if errors.As(err, &overlap) {
			return writeError(c, fiber.StatusUnprocessableEntity, "TERM_DATE_OVERLAP",
				overlap.Error(), fiber.Map{
					"conflicting_term": overlap.ConflictingName,
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
		"academic_year_id": term.AcademicYearID,
		"version":          term.Version,
	}

	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}

	return c.JSON(resp)
}

// DeleteTerm handles DELETE /api/v1/academic-terms/:id.
func (h *Handler) DeleteTerm(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	id := c.Params("id")
	schoolID := c.Locals("school_id").(string)

	err := h.svc.DeleteTerm(c.Context(), id, tenantID, schoolID, userID, nil)
	if err != nil {
		var hasDeps *HasDependentsError
		if errors.As(err, &hasDeps) {
			return writeError(c, fiber.StatusConflict, "HAS_DEPENDENTS", hasDeps.Message, nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
