package students

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes student HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts student routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	students := router.Group("/api/v1/students")
	students.Get("/list", middleware.RequireAuth, h.List)
}

// schoolIDFromContext extracts the school ID from the request context.
func schoolIDFromContext(c *fiber.Ctx) string {
	if schoolID := c.Query("school_id"); schoolID != "" {
		return schoolID
	}
	if schoolID, ok := c.Locals("active_school_id").(string); ok && schoolID != "" {
		return schoolID
	}
	return ""
}

// List handles GET /api/v1/students/list.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := schoolIDFromContext(c)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	filter := ListFilter{
		TenantID: tenantID,
		SchoolID: schoolID,
		Page:     page,
		Limit:    limit,
		Search:   c.Query("search"),
		ClassID:  c.Query("class_id"),
		Gender:   c.Query("gender"),
	}

	result, err := h.svc.ListStudents(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}
