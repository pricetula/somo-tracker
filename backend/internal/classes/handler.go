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
	schools.Get("/classes/grades", h.requireAuth, h.ListGrades)
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
// Supports optional query params: grade_ids (comma-separated), search, is_active.
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

	// Parse filter params
	params := ListClassesParams{
		Search: c.Query("search", ""),
	}

	if gradeIDs := c.Query("grade_ids", ""); gradeIDs != "" {
		params.GradeIDs = splitAndTrim(gradeIDs, ",")
	}

	if isActiveStr := c.Query("is_active", ""); isActiveStr != "" {
		isActive := isActiveStr == "true"
		params.IsActive = &isActive
	}

	classes, err := h.svc.ListClasses(c.Context(), schoolID, tenantID, params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(classes)
}

// ListGrades handles GET /api/v1/schools/classes/grades.
// Returns all grade records for the school's education system.
func (h *Handler) ListGrades(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.svc.ResolveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve school: " + err.Error(),
		})
	}

	grades, err := h.svc.ListGrades(c.Context(), schoolID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(grades)
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
// splitAndTrim splits a string by a separator and trims whitespace from each element.
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			part := trimSpace(s[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	part := trimSpace(s[start:])
	if part != "" {
		result = append(result, part)
	}
	return result
}

// trimSpace strips leading and trailing whitespace from a string.
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

var Module = fx.Module("classes",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
