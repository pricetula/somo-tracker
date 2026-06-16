package classes

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
)

// Handler exposes class-management HTTP endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
	log     *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, authSvc *auth.Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, authSvc: authSvc, log: log}
}

// RegisterRoutes mounts class routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Mount under /api/v1/schools
	schools := router.Group("/api/v1/schools")

	schools.Get("/classes", h.requireAuth, h.ListClasses)
	schools.Post("/classes/generate", h.requireAuth, h.GenerateClasses)
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

// ListClasses handles GET /api/v1/schools/classes.
// Returns all active classes for the authenticated school's current academic year.
func (h *Handler) ListClasses(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.svc.ResolveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve school: " + err.Error(),
		})
	}

	classes, err := h.svc.ListClasses(c.Context(), schoolID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(classes)
}

// GenerateClasses handles POST /api/v1/schools/classes/generate.
// Accepts a list of stream names, cross-multiplies them with the school's
// grade levels, and bulk-inserts all resulting classrooms in a single transaction.
func (h *Handler) GenerateClasses(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var payload GeneratePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	if len(payload.Streams) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "at least one stream name is required",
		})
	}

	// Validate stream names are non-empty
	for _, s := range payload.Streams {
		if s == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
				Error:   "invalid_input",
				Message: "stream names must not be empty",
			})
		}
	}

	schoolID, err := h.svc.ResolveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve school: " + err.Error(),
		})
	}

	result, err := h.svc.GenerateClasses(c.Context(), schoolID, tenantID, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// Module is an fx-compatible module for the classes domain.
var Module = fx.Module("classes",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
