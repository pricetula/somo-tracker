package curriculum

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
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
	areas.Delete("/:id", middleware.RequireAuth, h.DeleteLearningArea)

	// Strands
	strands := router.Group("/api/v1/curriculum/strands")
	strands.Post("/", middleware.RequireAuth, h.CreateStrand)
	strands.Get("/", middleware.RequireAuth, h.ListStrands)
	strands.Put("/:id", middleware.RequireAuth, h.UpdateStrand)
	strands.Delete("/:id", middleware.RequireAuth, h.DeleteStrand)

	// Sub-Strands
	subStrands := router.Group("/api/v1/curriculum/sub-strands")
	subStrands.Post("/", middleware.RequireAuth, h.CreateSubStrand)
	subStrands.Get("/", middleware.RequireAuth, h.ListSubStrands)
	subStrands.Put("/:id", middleware.RequireAuth, h.UpdateSubStrand)
	subStrands.Delete("/:id", middleware.RequireAuth, h.DeleteSubStrand)

	// Performance Indicators
	indicators := router.Group("/api/v1/curriculum/performance-indicators")
	indicators.Post("/", middleware.RequireAuth, h.CreatePerformanceIndicator)
	indicators.Get("/", middleware.RequireAuth, h.ListPerformanceIndicators)
	indicators.Put("/:id", middleware.RequireAuth, h.UpdatePerformanceIndicator)
	indicators.Delete("/:id", middleware.RequireAuth, h.DeletePerformanceIndicator)
}

// ── Helpers ──────────────────────────────────────────────────────────────

func getTenantAndSchool(c *fiber.Ctx) (string, string, error) {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "", "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "authentication required",
		})
	}
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}
	return tenantID, schoolID, nil
}

func invalidBody(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
		"code":    "invalid_input",
		"message": "invalid request body",
	})
}

// ── Learning Area Handlers ───────────────────────────────────────────────

// CreateLearningArea handles POST /api/v1/curriculum/learning-areas.
func (h *Handler) CreateLearningArea(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreateLearningAreaPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	id, err := h.svc.CreateLearningArea(c.Context(), CreateLearningAreaParams{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		Name:           payload.Name,
		Code:           payload.Code,
		EducationLevel: payload.EducationLevel,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListLearningAreas handles GET /api/v1/curriculum/learning-areas.
func (h *Handler) ListLearningAreas(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var educationLevel *string
	if el := c.Query("education_level"); el != "" {
		educationLevel = &el
	}

	areas, err := h.svc.ListLearningAreas(c.Context(), tenantID, schoolID, educationLevel)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListLearningAreasResponse{
		LearningAreas: areas,
		Total:         len(areas),
	})
}

// GetLearningAreaByID handles GET /api/v1/curriculum/learning-areas/:id.
func (h *Handler) GetLearningAreaByID(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	areaID := c.Params("id")
	if areaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "learning area id is required",
		})
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
		return err
	}

	areaID := c.Params("id")
	if areaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "learning area id is required",
		})
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
		return err
	}

	areaID := c.Params("id")
	if areaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "learning area id is required",
		})
	}

	var payload UpdateLearningAreaPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
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

	if err := h.svc.UpdateLearningArea(c.Context(), params); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteLearningArea handles DELETE /api/v1/curriculum/learning-areas/:id.
func (h *Handler) DeleteLearningArea(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	areaID := c.Params("id")
	if areaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "learning area id is required",
		})
	}

	if err := h.svc.DeleteLearningArea(c.Context(), areaID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ── Strand Handlers ──────────────────────────────────────────────────────

// CreateStrand handles POST /api/v1/curriculum/strands.
func (h *Handler) CreateStrand(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreateStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	// The service verifies learning_area_id belongs to this tenant/school
	id, err := h.svc.CreateStrand(c.Context(), CreateStrandParams(payload), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListStrands handles GET /api/v1/curriculum/strands?learning_area_id=X.
func (h *Handler) ListStrands(c *fiber.Ctx) error {
	learningAreaID := c.Query("learning_area_id")
	if learningAreaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "learning_area_id query parameter is required",
		})
	}

	strands, err := h.svc.ListStrands(c.Context(), learningAreaID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListStrandsResponse{
		Strands: strands,
		Total:   len(strands),
	})
}

// UpdateStrand handles PUT /api/v1/curriculum/strands/:id.
func (h *Handler) UpdateStrand(c *fiber.Ctx) error {
	strandID := c.Params("id")
	if strandID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "strand id is required",
		})
	}

	var payload UpdateStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	if err := h.svc.UpdateStrand(c.Context(), UpdateStrandParams{
		ID:   strandID,
		Name: payload.Name,
	}); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteStrand handles DELETE /api/v1/curriculum/strands/:id.
func (h *Handler) DeleteStrand(c *fiber.Ctx) error {
	strandID := c.Params("id")
	if strandID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "strand id is required",
		})
	}

	if err := h.svc.DeleteStrand(c.Context(), strandID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ── Sub-Strand Handlers ──────────────────────────────────────────────────

// CreateSubStrand handles POST /api/v1/curriculum/sub-strands.
func (h *Handler) CreateSubStrand(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreateSubStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	id, err := h.svc.CreateSubStrand(c.Context(), CreateSubStrandParams(payload), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListSubStrands handles GET /api/v1/curriculum/sub-strands?strand_id=X.
func (h *Handler) ListSubStrands(c *fiber.Ctx) error {
	strandID := c.Query("strand_id")
	if strandID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "strand_id query parameter is required",
		})
	}

	subs, err := h.svc.ListSubStrands(c.Context(), strandID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListSubStrandsResponse{
		SubStrands: subs,
		Total:      len(subs),
	})
}

// UpdateSubStrand handles PUT /api/v1/curriculum/sub-strands/:id.
func (h *Handler) UpdateSubStrand(c *fiber.Ctx) error {
	subID := c.Params("id")
	if subID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "sub-strand id is required",
		})
	}

	var payload UpdateSubStrandPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	if err := h.svc.UpdateSubStrand(c.Context(), UpdateSubStrandParams{
		ID:   subID,
		Name: payload.Name,
	}); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteSubStrand handles DELETE /api/v1/curriculum/sub-strands/:id.
func (h *Handler) DeleteSubStrand(c *fiber.Ctx) error {
	subID := c.Params("id")
	if subID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "sub-strand id is required",
		})
	}

	if err := h.svc.DeleteSubStrand(c.Context(), subID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ── Performance Indicator Handlers ───────────────────────────────────────

// CreatePerformanceIndicator handles POST /api/v1/curriculum/performance-indicators.
func (h *Handler) CreatePerformanceIndicator(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreatePerformanceIndicatorPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	id, err := h.svc.CreatePerformanceIndicator(c.Context(), CreatePerformanceIndicatorParams(payload), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListPerformanceIndicators handles GET /api/v1/curriculum/performance-indicators?sub_strand_id=X.
func (h *Handler) ListPerformanceIndicators(c *fiber.Ctx) error {
	subStrandID := c.Query("sub_strand_id")
	if subStrandID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "sub_strand_id query parameter is required",
		})
	}

	indicators, err := h.svc.ListPerformanceIndicators(c.Context(), subStrandID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListPerformanceIndicatorsResponse{
		PerformanceIndicators: indicators,
		Total:                 len(indicators),
	})
}

// UpdatePerformanceIndicator handles PUT /api/v1/curriculum/performance-indicators/:id.
func (h *Handler) UpdatePerformanceIndicator(c *fiber.Ctx) error {
	indicatorID := c.Params("id")
	if indicatorID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "performance indicator id is required",
		})
	}

	var payload UpdatePerformanceIndicatorPayload
	if err := c.BodyParser(&payload); err != nil {
		return invalidBody(c)
	}

	if err := h.svc.UpdatePerformanceIndicator(c.Context(), UpdatePerformanceIndicatorParams{
		ID:            indicatorID,
		Description:   payload.Description,
		SequenceOrder: payload.SequenceOrder,
	}); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeletePerformanceIndicator handles DELETE /api/v1/curriculum/performance-indicators/:id.
func (h *Handler) DeletePerformanceIndicator(c *fiber.Ctx) error {
	indicatorID := c.Params("id")
	if indicatorID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "performance indicator id is required",
		})
	}

	if err := h.svc.DeletePerformanceIndicator(c.Context(), indicatorID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
