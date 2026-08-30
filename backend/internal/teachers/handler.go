package teachers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// Handler exposes teacher HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts teacher routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	teachers := router.Group("/api/v1/teachers")
	teachers.Get("/", middleware.RequireAuth, h.List)
	teachers.Get("/:user_id", middleware.RequireAuth, h.GetByID)
	teachers.Put("/:user_id", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.Update)
	teachers.Patch("/:user_id/active", middleware.RequireAuth, h.ToggleActive)
	teachers.Delete("/", middleware.RequireAuth, h.Delete)

	// Teacher-specific dashboard routes
	teachers.Get("/:user_id/classes", middleware.RequireAuth, h.ListClasses)
	teachers.Get("/:user_id/timetable", middleware.RequireAuth, h.GetTimetable)
	teachers.Get("/:user_id/lessons", middleware.RequireAuth, h.GetLessons)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/teachers
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := strings.TrimSpace(c.Query("search", ""))
	includeInactive := strings.ToLower(c.Query("include_inactive", "false")) == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	teachersList, total, err := h.svc.ListTeachers(c.Context(), tenantID, schoolID, includeInactive, offset, limit, search)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListResponse{
		Items: teachersList,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ToggleActive handles PATCH /api/v1/teachers/:user_id/active
func (h *Handler) ToggleActive(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Params("user_id")

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var req ToggleActiveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.ToggleActive(c.Context(), tenantID, schoolID, userID, req.IsActive); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "teacher status updated",
	})
}

// GetByID handles GET /api/v1/teachers/:user_id
func (h *Handler) GetByID(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Params("user_id")

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	teacher, err := h.svc.GetTeacherByID(c.Context(), userID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(teacher)
}

// Update handles PUT /api/v1/teachers/:user_id
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Params("user_id")

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var payload UpdateTeacherPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UpdateTeacher(c.Context(), userID, tenantID, schoolID, payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "teacher updated",
	})
}

// Delete handles DELETE /api/v1/teachers/:user_id
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var payload struct {
		UserID string `json:"user_id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "malformed request body",
		})
	}
	if payload.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "user_id is required",
		})
	}

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	if err := h.svc.Delete(c.Context(), tenantID, schoolID, payload.UserID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "teacher deleted",
	})
}

// ─── Teacher Dashboard Handlers ───────────────────────────────────────────

// ListClasses handles GET /api/v1/teachers/:user_id/classes.
func (h *Handler) ListClasses(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	userID := c.Params("user_id")
	termID := c.Query("term_id")

	result, err := h.svc.ListTeacherClasses(c.Context(), tenantID, schoolID, userID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetTimetable handles GET /api/v1/teachers/:user_id/timetable?day_of_week=1.
// GetLessons handles GET /api/v1/teachers/:user_id/lessons.
func (h *Handler) GetLessons(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	userID := c.Params("user_id")
	weekStart := c.Query("week_start", "")
	limitStr := c.Query("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		limit = 20
	}

	result, err := h.svc.ListTeacherLessonTimeline(c.Context(), tenantID, schoolID, userID, weekStart, limit)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

func (h *Handler) GetTimetable(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	userID := c.Params("user_id")
	dayOfWeek, err := strconv.Atoi(c.Query("day_of_week", "1"))
	if err != nil || dayOfWeek < 1 || dayOfWeek > 7 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "day_of_week must be 1 (Monday) through 7 (Sunday)",
		})
	}

	result, err := h.svc.GetTeacherTimetable(c.Context(), tenantID, schoolID, userID, dayOfWeek)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// Module is an fx-compatible module for the teachers domain.
var Module = fx.Module("teachers",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
