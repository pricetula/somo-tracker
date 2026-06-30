package activeschool

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/middleware"
)

const somoSchoolIDCookieName = "somo_school_id"

// Handler exposes active-school HTTP endpoints.
type Handler struct {
	svc *Service
	cfg config.Config
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, cfg config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// RegisterRoutes mounts active-school routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	as := router.Group("/api/v1/active-school")
	as.Put("/", middleware.RequireAuth, h.Switch)
	as.Get("/", middleware.RequireAuth, h.Get)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// Switch handles PUT /api/v1/active-school.
// Updates the active school for the authenticated user (upsert) and updates
// the somo_school_id cookie so subsequent requests pick up the new school.
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

	// Update the school ID cookie so the security middleware picks up the new
	// active school on the next request.
	c.Cookie(&fiber.Cookie{
		Name:     somoSchoolIDCookieName,
		Value:    payload.SchoolID,
		HTTPOnly: false,
		Secure:   h.cfg.AppEnv != "development",
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   2592000, // 30 days
	})

	return c.JSON(fiber.Map{
		"message":   "active school updated",
		"school_id": payload.SchoolID,
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
