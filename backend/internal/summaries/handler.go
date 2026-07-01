package summaries

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler
// ============================================================================

// Handler exposes competency summary HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts summary routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	summaries := router.Group("/api/v1/summaries")
	summaries.Get("/", middleware.RequireAuth, h.ListSummaries)
	summaries.Get("/:id", middleware.RequireAuth, h.GetSummary)
	summaries.Put("/:id/override", middleware.RequireAuth, h.SetOverrideLevel)
	summaries.Post("/calculate", middleware.RequireAuth, h.CalculateSummaries)
	summaries.Post("/calculate-for-class", middleware.RequireAuth, h.CalculateForClass)
	summaries.Post("/:id/mark-synced", middleware.RequireAuth, h.MarkSynced)
}

// ============================================================================
// Error response helper — matches the canonical { code, message, errors } shape
// ============================================================================

type errorResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string, fieldErrors map[string][]string) error {
	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
		Errors:  fieldErrors,
	})
}

// ============================================================================
// LIST
// ============================================================================

// ListSummaries handles GET /api/v1/summaries.
func (h *Handler) ListSummaries(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	query := ListSummariesQuery{
		StudentID:      c.Query("student_id"),
		ClassID:        c.Query("class_id"),
		LearningAreaID: c.Query("learning_area_id"),
	}

	if termStr := c.Query("term"); termStr != "" {
		term, err := strconv.Atoi(termStr)
		if err == nil {
			query.Term = term
		}
	}

	if yearStr := c.Query("academic_year"); yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err == nil {
			query.AcademicYear = year
		}
	}

	summaries, err := h.svc.List(c.Context(), tenantID, query)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListSummariesResponse{Data: summaries})
}

// ============================================================================
// GET BY ID
// ============================================================================

// GetSummary handles GET /api/v1/summaries/:id.
func (h *Handler) GetSummary(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "summary id is required", nil)
	}

	summary, err := h.svc.GetByID(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(CompetencySummaryDetailResponse{Data: *summary})
}

// ============================================================================
// OVERRIDE
// ============================================================================

// SetOverrideLevel handles PUT /api/v1/summaries/:id/override.
func (h *Handler) SetOverrideLevel(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "summary id is required", nil)
	}

	var body OverrideLevelPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if err := h.svc.SetOverrideLevel(c.Context(), id, tenantID, body); err != nil {
		if errors.Is(err, ErrInvalidRubricLevel) {
			return writeError(c, fiber.StatusBadRequest, "invalid_rubric_level",
				"Invalid rubric level. Valid values: EE, ME, AE, BE, EE1, EE2, ME1, ME2, AE1, AE2, BE1, BE2", nil)
		}
		if errors.Is(err, ErrConflict) {
			return writeError(c, fiber.StatusConflict, "already_synced",
				"Cannot override a summary that has already been synced to KNEC.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// CALCULATE (legacy endpoint — single class)
// ============================================================================

// CalculateSummaries handles POST /api/v1/summaries/calculate.
// This is a convenience endpoint that expects the payload to include class_id.
func (h *Handler) CalculateSummaries(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var body CalculateForClassPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.ClassID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "class_id is required", nil)
	}
	if body.AcademicYear == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "academic_year is required", nil)
	}
	if body.Term == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "term is required", nil)
	}

	count, err := h.svc.CalculateForClass(c.Context(), tenantID, body)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(CalculateResponse{Count: count})
}

// ============================================================================
// CALCULATE FOR CLASS
// ============================================================================

// CalculateForClass handles POST /api/v1/summaries/calculate-for-class.
func (h *Handler) CalculateForClass(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var body CalculateForClassPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.ClassID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "class_id is required", nil)
	}
	if body.AcademicYear == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "academic_year is required", nil)
	}
	if body.Term == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "term is required", nil)
	}

	count, err := h.svc.CalculateForClass(c.Context(), tenantID, body)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(CalculateResponse{Count: count})
}

// ============================================================================
// MARK SYNCED
// ============================================================================

// MarkSynced handles POST /api/v1/summaries/:id/mark-synced.
func (h *Handler) MarkSynced(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "summary id is required", nil)
	}

	var body MarkSyncedPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if err := h.svc.MarkSynced(c.Context(), id, tenantID, body); err != nil {
		if errors.Is(err, ErrAlreadySynced) {
			return writeError(c, fiber.StatusConflict, "already_synced",
				"This summary has already been synced to KNEC and cannot be re-synced.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// fx Module
// ============================================================================

// Module is an fx-compatible module for the summaries domain.
var Module = fx.Module("summaries",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(Repository)),
		),
		NewService,
		NewHandler,
	),
)
