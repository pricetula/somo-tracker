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
	teachers.Patch("/:user_id/active", middleware.RequireAuth, h.ToggleActive)
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
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	search := strings.TrimSpace(c.Query("search", ""))
	includeInactive := strings.ToLower(c.Query("include_inactive", "false")) == "true"

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	teachersList, total, err := h.svc.ListTeachers(c.Context(), tenantID, schoolID, includeInactive, offset, perPage, search)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListResponse{
		Teachers: teachersList,
		Total:    total,
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

// Module is an fx-compatible module for the teachers domain.
var Module = fx.Module("teachers",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
