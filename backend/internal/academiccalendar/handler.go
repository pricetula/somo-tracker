package academiccalendar

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
)

// Handler exposes academic-calendar HTTP endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
	log     *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, authSvc *auth.Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, authSvc: authSvc, log: log}
}

// RegisterRoutes mounts academic-calendar routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Mount under /api/v1/schools
	schools := router.Group("/api/v1/schools")

	schools.Get("/current-calendar", h.requireAuth, h.GetCurrentCalendar)
	schools.Post("/current-calendar", h.requireAuth, h.UpsertCurrentCalendar)
}

// requireAuth extracts tenant_id from the session cookie and stores it in locals.
func (h *Handler) requireAuth(c *fiber.Ctx) error {
	token := c.Cookies("somo_sid")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "no session cookie found",
		})
	}

	session, err := h.authSvc.GetSession(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "invalid or expired session",
		})
	}

	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	return c.Next()
}

// GetCurrentCalendar handles GET /api/v1/schools/current-calendar.
func (h *Handler) GetCurrentCalendar(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.svc.ResolveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve school: " + err.Error(),
		})
	}

	cal, err := h.svc.GetCurrentCalendar(c.Context(), schoolID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}
	if cal == nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorBody{
			Error:   "not_found",
			Message: "no academic calendar configured yet",
		})
	}

	return c.JSON(cal)
}

// UpsertCurrentCalendar handles POST /api/v1/schools/current-calendar.
func (h *Handler) UpsertCurrentCalendar(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var payload SavePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	if payload.Year == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "year is required",
		})
	}
	if len(payload.Periods) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "at least one period is required",
		})
	}

	schoolID, err := h.svc.ResolveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve school: " + err.Error(),
		})
	}

	cal, err := h.svc.SaveCurrentCalendar(c.Context(), schoolID, tenantID, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(cal)
}

// Module is an fx-compatible module for the academic calendar domain.
var Module = fx.Module("academiccalendar",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
