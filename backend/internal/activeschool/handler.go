package activeschool

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes active-school HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts active-school routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	as := router.Group("/api/v1/active-school")
	as.Put("/", h.requireAuth, h.Switch)
	as.Get("/", h.requireAuth, h.Get)
}

// ─── Auth middleware ───────────────────────────────────────────────────────

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
	return c.Next()
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// Switch handles PUT /api/v1/active-school.
// Updates the active school for the authenticated user (upsert).
func (h *Handler) Switch(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var payload SwitchActiveSchoolPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if payload.SchoolID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id is required",
		})
	}

	if err := h.svc.SwitchActiveSchool(c.Context(), tenantID, userID, payload.SchoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "active school updated",
	})
}

// Get handles GET /api/v1/active-school.
// Returns the active school ID for the authenticated user.
func (h *Handler) Get(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.svc.GetActiveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"school_id": schoolID,
	})
}
