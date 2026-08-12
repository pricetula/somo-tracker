package curriculum

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
	"somotracker/backend/internal/xerrors"
)

// Handler exposes curriculum HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all curriculum routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Learning Areas
	areas := router.Group("/api/v1/curriculum/learning-areas")
	areas.Post("/", middleware.RequireAuth, h.CreateLearningArea)
	areas.Get("/", middleware.RequireAuth, h.ListLearningAreas)
	areas.Get("/:id", middleware.RequireAuth, h.GetLearningAreaByID)
	areas.Get("/:id/tree", middleware.RequireAuth, h.GetTree)
	areas.Put("/:id", middleware.RequireAuth, h.UpdateLearningArea)
	areas.Delete("/", middleware.RequireAuth, h.DeleteLearningArea)

	// Strands
	strands := router.Group("/api/v1/curriculum/strands")
	strands.Post("/", middleware.RequireAuth, h.CreateStrand)
	strands.Get("/", middleware.RequireAuth, h.ListStrands)
	strands.Put("/:id", middleware.RequireAuth, h.UpdateStrand)
	strands.Delete("/", middleware.RequireAuth, h.DeleteStrand)

	// Sub-Strands
	subStrands := router.Group("/api/v1/curriculum/sub-strands")
	subStrands.Post("/", middleware.RequireAuth, h.CreateSubStrand)
	subStrands.Get("/", middleware.RequireAuth, h.ListSubStrands)
	subStrands.Put("/:id", middleware.RequireAuth, h.UpdateSubStrand)
	subStrands.Delete("/", middleware.RequireAuth, h.DeleteSubStrand)

	// Performance Indicators
	indicators := router.Group("/api/v1/curriculum/performance-indicators")
	indicators.Post("/", middleware.RequireAuth, h.CreatePerformanceIndicator)
	indicators.Get("/", middleware.RequireAuth, h.ListPerformanceIndicators)
	indicators.Put("/:id", middleware.RequireAuth, h.UpdatePerformanceIndicator)
	indicators.Delete("/", middleware.RequireAuth, h.DeletePerformanceIndicator)
}

// ── Helpers ──────────────────────────────────────────────────────────────

func getTenantAndSchool(c *fiber.Ctx) (string, string, error) {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "", "", fmt.Errorf("authentication required: %w", xerrors.ErrUnauthorized)
	}
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", fmt.Errorf("active school not set: %w", xerrors.ErrInvalidInput)
	}
	return tenantID, schoolID, nil
}

func invalidBody(c *fiber.Ctx) error {
	return fmt.Errorf("invalid request body: %w", xerrors.UnprocessableEntity("malformed request body"))
}

// parseRepeatedQuery reads all values for a given query parameter name.
// Supports two styles:
//   - repeated params: ?grade_level=G4&grade_level=G5
//   - comma-separated: ?grade_level=G4,G5
func parseRepeatedQuery(c *fiber.Ctx, name string) []string {
	// Check for repeated query params first (e.g., ?grade=G4&grade=G5)
	all := c.Request().URI().QueryArgs().PeekMulti(name)
	if len(all) > 1 {
		result := make([]string, 0, len(all))
		for _, v := range all {
			s := strings.TrimSpace(string(v))
			if s != "" {
				result = append(result, s)
			}
		}
		return result
	}

	// Single value (or none) — could be comma-separated for client convenience
	// (e.g., ?grade_level=G4,G5). c.Query returns only the first value, which
	// is correct here since we already know there's at most one occurrence.
	vals := c.Query(name, "")
	if vals == "" {
		return nil
	}

	parts := strings.Split(vals, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ── Learning Area Handlers ───────────────────────────────────────────────

// CreateLearningArea handles POST /api/v1/curriculum/learning-areas.
func (h *Handler) CreateLearningArea(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	var payload CreateLearningAreaPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	id, err := h.svc.CreateLearningArea(c.Context(), CreateLearningAreaParams{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		Name:           payload.Name,
		Code:           payload.Code,
		EducationLevel: payload.EducationLevel,
		GradeLevel:     payload.GradeLevel,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListLearningAreas handles GET /api/v1/curriculum/learning-areas.
// Supports optional multi-select filtering via repeated query params:
//
//	?education_level=Early_Years&education_level=Upper_Primary
//	&grade_level=PP1&grade_level=G4
func (h *Handler) ListLearningAreas(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// Multi-select filters — parse repeated query params
	educationLevels := parseRepeatedQuery(c, "education_level")
	gradeLevels := parseRepeatedQuery(c, "grade_level")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := strings.TrimSpace(c.Query("search"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	areas, total, err := h.svc.ListLearningAreas(c.Context(), tenantID, schoolID, educationLevels, gradeLevels, search, page, limit)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListLearningAreasResponse{
		Items: areas,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetLearningAreaByID handles GET /api/v1/curriculum/learning-areas/:id.
func (h *Handler) GetLearningAreaByID(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	areaID := c.Params("id")
	if areaID == "" {
		return middleware.HTTPError(c, fmt.Errorf("learning area id is required: %w", xerrors.ErrInvalidInput))
	}

	area, err := h.svc.GetLearningArea(c.Context(), areaID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(area)
}

// GetTree handles GET /api/v1/curriculum/learning-areas/:id/tree.
func (h *Handler) GetTree(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	areaID := c.Params("id")
	if areaID == "" {
		return middleware.HTTPError(c, fmt.Errorf("learning area id is required: %w", xerrors.ErrInvalidInput))
	}

	tree, err := h.svc.GetTree(c.Context(), areaID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(tree)
}

// UpdateLearningArea handles PUT /api/v1/curriculum/learning-areas/:id.
func (h *Handler) UpdateLearningArea(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	areaID := c.Params("id")
	if areaID == "" {
		return middleware.HTTPError(c, fmt.Errorf("learning area id is required: %w", xerrors.ErrInvalidInput))
	}

	var payload UpdateLearningAreaPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	// Verify the learning area exists and belongs to this tenant/school
	if _, err := h.svc.GetLearningArea(c.Context(), areaID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	params := UpdateLearningAreaParams{
		ID:       areaID,
		TenantID: tenantID,
		SchoolID: schoolID,
	}
	if payload.Name != nil {
		params.Name = payload.Name
	}
	if payload.Code != nil {
		params.Code = payload.Code
	}
	if payload.EducationLevel != nil {
		params.EducationLevel = payload.EducationLevel
	}
	if payload.GradeLevel != nil {
		params.GradeLevel = payload.GradeLevel
	}

	if err := h.svc.UpdateLearningArea(c.Context(), params); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteLearningArea handles DELETE /api/v1/curriculum/learning-areas/:id.
func (h *Handler) DeleteLearningArea(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}
	if payload.ID == "" {
		return middleware.HTTPError(c, fmt.Errorf("learning area id is required: %w", xerrors.ErrInvalidInput))
	}

	if err := h.svc.DeleteLearningArea(c.Context(), payload.ID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ── Strand Handlers ──────────────────────────────────────────────────────

// CreateStrand handles POST /api/v1/curriculum/strands.
func (h *Handler) CreateStrand(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	var payload CreateStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	// The service verifies learning_area_id belongs to this tenant/school
	id, err := h.svc.CreateStrand(c.Context(), CreateStrandParams{
		LearningAreaID: payload.LearningAreaID,
		Name:           payload.Name,
	}, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListStrands handles GET /api/v1/curriculum/strands?learning_area_id=X.
func (h *Handler) ListStrands(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	learningAreaID := c.Query("learning_area_id")
	if learningAreaID == "" {
		return middleware.HTTPError(c, fmt.Errorf("learning_area_id query parameter is required: %w", xerrors.ErrInvalidInput))
	}

	strands, err := h.svc.ListStrands(c.Context(), learningAreaID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListStrandsResponse{
		Items: strands,
		Total: len(strands),
		Page:  1,
		Limit: len(strands),
	})
}

// UpdateStrand handles PUT /api/v1/curriculum/strands/:id.
func (h *Handler) UpdateStrand(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	strandID := c.Params("id")
	if strandID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "strand id is required",
		})
	}

	var payload UpdateStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	if err := h.svc.UpdateStrand(c.Context(), strandID, tenantID, payload.Name); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteStrand handles DELETE /api/v1/curriculum/strands/:id.
func (h *Handler) DeleteStrand(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "strand id is required",
		})
	}

	if err := h.svc.DeleteStrand(c.Context(), payload.ID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ── Sub-Strand Handlers ──────────────────────────────────────────────────

// CreateSubStrand handles POST /api/v1/curriculum/sub-strands.
func (h *Handler) CreateSubStrand(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	var payload CreateSubStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	id, err := h.svc.CreateSubStrand(c.Context(), CreateSubStrandParams{
		StrandID: payload.StrandID,
		Name:     payload.Name,
	}, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListSubStrands handles GET /api/v1/curriculum/sub-strands?strand_id=X.
func (h *Handler) ListSubStrands(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	strandID := c.Query("strand_id")
	if strandID == "" {
		return middleware.HTTPError(c, fmt.Errorf("strand_id query parameter is required: %w", xerrors.ErrInvalidInput))
	}

	subs, err := h.svc.ListSubStrands(c.Context(), strandID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListSubStrandsResponse{
		Items: subs,
		Total: len(subs),
		Page:  1,
		Limit: len(subs),
	})
}

// UpdateSubStrand handles PUT /api/v1/curriculum/sub-strands/:id.
func (h *Handler) UpdateSubStrand(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	subID := c.Params("id")
	if subID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "sub-strand id is required",
		})
	}

	var payload UpdateSubStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	if err := h.svc.UpdateSubStrand(c.Context(), subID, tenantID, payload.Name); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteSubStrand handles DELETE /api/v1/curriculum/sub-strands/:id.
func (h *Handler) DeleteSubStrand(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "sub-strand id is required",
		})
	}

	if err := h.svc.DeleteSubStrand(c.Context(), payload.ID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ── Performance Indicator Handlers ───────────────────────────────────────

// CreatePerformanceIndicator handles POST /api/v1/curriculum/performance-indicators.
func (h *Handler) CreatePerformanceIndicator(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	var payload CreatePerformanceIndicatorPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	id, err := h.svc.CreatePerformanceIndicator(c.Context(), CreatePerformanceIndicatorParams{
		SubStrandID:   payload.SubStrandID,
		Description:   payload.Description,
		SequenceOrder: payload.SequenceOrder,
	}, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListPerformanceIndicators handles GET /api/v1/curriculum/performance-indicators?sub_strand_id=X.
func (h *Handler) ListPerformanceIndicators(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	subStrandID := c.Query("sub_strand_id")
	if subStrandID == "" {
		return middleware.HTTPError(c, fmt.Errorf("sub_strand_id query parameter is required: %w", xerrors.ErrInvalidInput))
	}

	indicators, err := h.svc.ListPerformanceIndicators(c.Context(), subStrandID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListPerformanceIndicatorsResponse{
		Items: indicators,
		Total: len(indicators),
		Page:  1,
		Limit: len(indicators),
	})
}

// UpdatePerformanceIndicator handles PUT /api/v1/curriculum/performance-indicators/:id.
func (h *Handler) UpdatePerformanceIndicator(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	indicatorID := c.Params("id")
	if indicatorID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "performance indicator id is required",
		})
	}

	var payload UpdatePerformanceIndicatorPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}

	if err := h.svc.UpdatePerformanceIndicator(c.Context(), indicatorID, tenantID, payload.Description, payload.SequenceOrder); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeletePerformanceIndicator handles DELETE /api/v1/curriculum/performance-indicators/:id.
func (h *Handler) DeletePerformanceIndicator(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, invalidBody(c))
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "performance indicator id is required",
		})
	}

	if err := h.svc.DeletePerformanceIndicator(c.Context(), payload.ID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
