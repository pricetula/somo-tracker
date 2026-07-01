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

// RegisterRoutes mounts curriculum routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	areas := router.Group("/api/v1/curriculum/learning-areas")
	areas.Post("/", middleware.RequireAuth, h.Create)
	areas.Get("/", middleware.RequireAuth, h.List)
	areas.Get("/:id", middleware.RequireAuth, h.GetByID)
	areas.Put("/:id", middleware.RequireAuth, h.Update)
	areas.Delete("/:id", middleware.RequireAuth, h.Delete)
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

// ── Handlers ────────────────────────────────────────────────────────────

// Create handles POST /api/v1/curriculum/learning-areas.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreateLearningAreaPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
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

// List handles GET /api/v1/curriculum/learning-areas.
func (h *Handler) List(c *fiber.Ctx) error {
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

// GetByID handles GET /api/v1/curriculum/learning-areas/:id.
func (h *Handler) GetByID(c *fiber.Ctx) error {
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

// Update handles PUT /api/v1/curriculum/learning-areas/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
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
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
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

// Delete handles DELETE /api/v1/curriculum/learning-areas/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
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
