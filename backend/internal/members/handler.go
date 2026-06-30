package members

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// Handler exposes member HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts member routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	members := router.Group("/api/v1/members")
	members.Get("/", middleware.RequireAuth, h.List)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/members?role=TEACHER
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	role := strings.TrimSpace(c.Query("role", ""))
	if role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "role query parameter is required (TEACHER, NURSE, FINANCE, or SCHOOL_ADMIN)",
		})
	}
	validRoles := map[string]bool{"TEACHER": true, "NURSE": true, "FINANCE": true, "SCHOOL_ADMIN": true}
	if !validRoles[role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "role must be TEACHER, NURSE, FINANCE, or SCHOOL_ADMIN",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	search := strings.TrimSpace(c.Query("search", ""))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	membersList, total, err := h.svc.ListMembers(c.Context(), tenantID, schoolID, role, offset, perPage, search)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListResponse{
		Members: membersList,
		Total:   total,
	})
}

// Module is an fx-compatible module for the members domain.
var Module = fx.Module("members",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
