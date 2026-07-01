package assessment

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

// Handler exposes assessment HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts assessment routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Blueprints
	blueprints := router.Group("/api/v1/assessment/blueprints")
	blueprints.Post("/", middleware.RequireAuth, h.CreateBlueprint)
	blueprints.Get("/", middleware.RequireAuth, h.ListBlueprints)
	blueprints.Get("/:id", middleware.RequireAuth, h.GetBlueprintDetail)
	blueprints.Put("/:id", middleware.RequireAuth, h.UpdateBlueprint)
	blueprints.Delete("/:id", middleware.RequireAuth, h.DeleteBlueprint)

	// Blueprint ↔ Indicator Linking
	blueprints.Post("/:id/indicators", middleware.RequireAuth, h.LinkIndicators)
	blueprints.Delete("/:id/indicators/:indicator_id", middleware.RequireAuth, h.UnlinkIndicator)

	// Sessions
	sessions := router.Group("/api/v1/assessment/sessions")
	sessions.Post("/", middleware.RequireAuth, h.CreateSession)
	sessions.Get("/", middleware.RequireAuth, h.ListSessions)
	sessions.Get("/:id", middleware.RequireAuth, h.GetSessionDetail)
	sessions.Put("/:id", middleware.RequireAuth, h.UpdateSession)
	sessions.Delete("/:id", middleware.RequireAuth, h.DeleteSession)

	// Results (nested under sessions)
	sessions.Post("/:id/results/batch", middleware.RequireAuth, h.BatchUpsertResults)
	sessions.Get("/:id/results", middleware.RequireAuth, h.ListResults)

	// Weight Configs (read-only, authenticated)
	weightConfigs := router.Group("/api/v1/assessment/weight-configs")
	weightConfigs.Get("/", middleware.RequireAuth, h.ListWeightConfigs)
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
// BLUEPRINTS
// ============================================================================

// CreateBlueprint handles POST /api/v1/assessment/blueprints.
func (h *Handler) CreateBlueprint(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	_ = userID // for future audit trail

	// school_id from active school context
	schoolID := c.Locals("school_id").(string)

	var body CreateBlueprintPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	bp, err := h.svc.CreateBlueprint(c.Context(), tenantID, schoolID, body)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return writeError(c, fiber.StatusConflict, "already_exists",
				"A blueprint with the same title, type, grade, year, term, and school already exists.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateBlueprintResponse{ID: bp.ID})
}

// ListBlueprints handles GET /api/v1/assessment/blueprints.
func (h *Handler) ListBlueprints(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)

	query := ListBlueprintsQuery{
		SchoolID:   c.Query("school_id", schoolID),
		GradeLevel: c.Query("grade_level"),
		Term:       0, // 0 means not set
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

	blueprints, err := h.svc.ListBlueprints(c.Context(), tenantID, query)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListBlueprintsResponse{Data: blueprints})
}

// GetBlueprintDetail handles GET /api/v1/assessment/blueprints/:id.
func (h *Handler) GetBlueprintDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)
	id := c.Params("id")

	detail, err := h.svc.GetBlueprintDetail(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(BlueprintDetailResponse{Data: *detail})
}

// UpdateBlueprint handles PUT /api/v1/assessment/blueprints/:id.
func (h *Handler) UpdateBlueprint(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)
	id := c.Params("id")

	var body UpdateBlueprintPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	// At least one field must be provided
	if body.Title == nil && body.Type == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "at least one of title or type must be provided", nil)
	}

	if err := h.svc.UpdateBlueprint(c.Context(), id, tenantID, schoolID, body); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return writeError(c, fiber.StatusConflict, "already_exists",
				"A blueprint with the same title, type, grade, year, term, and school already exists.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteBlueprint handles DELETE /api/v1/assessment/blueprints/:id.
func (h *Handler) DeleteBlueprint(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)
	id := c.Params("id")

	err := h.svc.DeleteBlueprint(c.Context(), id, tenantID, schoolID)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return writeError(c, fiber.StatusConflict, "referenced_by_sessions",
				"This blueprint has assessment sessions and cannot be deleted.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// SESSIONS
// ============================================================================

// CreateSession handles POST /api/v1/assessment/sessions.
func (h *Handler) CreateSession(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var body CreateSessionPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.BlueprintID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "blueprint_id is required", nil)
	}
	if body.ClassID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "class_id is required", nil)
	}
	if body.DateAdministered == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "date_administered is required", nil)
	}

	session, err := h.svc.CreateSession(c.Context(), tenantID, userID, body)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateSessionResponse{ID: session.ID})
}

// ListSessions handles GET /api/v1/assessment/sessions.
func (h *Handler) ListSessions(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	query := ListSessionsQuery{
		ClassID:     c.Query("class_id"),
		BlueprintID: c.Query("blueprint_id"),
	}

	sessions, err := h.svc.ListSessions(c.Context(), tenantID, query)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListSessionsResponse{Data: sessions})
}

// GetSessionDetail handles GET /api/v1/assessment/sessions/:id.
func (h *Handler) GetSessionDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	detail, err := h.svc.GetSessionDetail(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(SessionDetailResponse{Data: *detail})
}

// UpdateSession handles PUT /api/v1/assessment/sessions/:id.
func (h *Handler) UpdateSession(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	var body UpdateSessionPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	// At least one field must be provided
	if body.DateAdministered == nil && body.KNECUploadReference == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input",
			"at least one of date_administered or knec_upload_reference must be provided", nil)
	}

	if err := h.svc.UpdateSession(c.Context(), id, tenantID, body); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteSession handles DELETE /api/v1/assessment/sessions/:id.
func (h *Handler) DeleteSession(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if err := h.svc.DeleteSession(c.Context(), id, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// LEARNER RUBRIC RESULTS
// ============================================================================

// BatchUpsertResults handles POST /api/v1/assessment/sessions/:id/results/batch.
func (h *Handler) BatchUpsertResults(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	sessionID := c.Params("id")

	var body BatchUpsertResultsPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if len(body.Results) == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "at least one result is required", nil)
	}

	count, err := h.svc.BatchUpsertResults(c.Context(), sessionID, tenantID, body)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"rows_affected": count})
}

// ListResults handles GET /api/v1/assessment/sessions/:id/results.
func (h *Handler) ListResults(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	sessionID := c.Params("id")

	results, err := h.svc.ListResults(c.Context(), sessionID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListResultsResponse{Data: results})
}

// ============================================================================
// BLUEPRINT ↔ INDICATOR LINKING
// ============================================================================

// LinkIndicators handles POST /api/v1/assessment/blueprints/:id/indicators.
func (h *Handler) LinkIndicators(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)
	blueprintID := c.Params("id")

	var body LinkIndicatorPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if len(body.IndicatorIDs) == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "at least one indicator_id must be provided", nil)
	}

	err := h.svc.LinkIndicators(c.Context(), blueprintID, tenantID, schoolID, body)
	if err != nil {
		if errors.Is(err, ErrGradeLevelMismatch) {
			return writeError(c, fiber.StatusBadRequest, "grade_level_mismatch",
				"One or more indicators belong to a learning area that does not match the blueprint's grade level.", nil)
		}
		if errors.Is(err, ErrIndicatorLinked) {
			return writeError(c, fiber.StatusConflict, "indicator_already_linked",
				"One or more indicators are already linked to this blueprint.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// UnlinkIndicator handles DELETE /api/v1/assessment/blueprints/:id/indicators/:indicator_id.
func (h *Handler) UnlinkIndicator(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("school_id").(string)
	blueprintID := c.Params("id")
	indicatorID := c.Params("indicator_id")

	err := h.svc.UnlinkIndicator(c.Context(), blueprintID, indicatorID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// WEIGHT CONFIGS
// ============================================================================

// ListWeightConfigs handles GET /api/v1/assessment/weight-configs.
func (h *Handler) ListWeightConfigs(c *fiber.Ctx) error {
	query := ListWeightConfigsQuery{
		GradeLevel: c.Query("grade_level"),
		TargetExam: c.Query("target_exam"),
	}

	configs, err := h.svc.ListWeightConfigs(c.Context(), query)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListWeightConfigsResponse{Data: configs})
}

// ============================================================================
// fx Module
// ============================================================================

// Module is an fx-compatible module for the assessment domain.
var Module = fx.Module("assessment",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(Repository)),
			fx.As(new(ClassStudentResolver)),
		),
		NewService,
		NewHandler,
	),
)
