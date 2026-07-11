package cbcstreams

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes stream HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts stream routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	streams := router.Group("/api/v1/streams")
	streams.Get("/", middleware.RequireAuth, h.List)
}

// List handles GET /api/v1/streams.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	result, err := h.svc.ListStreams(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}
