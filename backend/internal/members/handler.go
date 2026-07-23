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
	members.Get("/:user_id", middleware.RequireAuth, h.GetByID)
	members.Put("/:user_id", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.Update)
	members.Patch("/:user_id/active", middleware.RequireAuth, h.ToggleActive)
	members.Delete("/", middleware.RequireAuth, h.Delete)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/members?role=TEACHER&include_inactive=true
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
	limit, _ := strconv.Atoi(c.Query("per_page", "50"))
	search := strings.TrimSpace(c.Query("search", ""))
	includeInactive := strings.ToLower(c.Query("include_inactive", "false")) == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var membersList []Member
	var total int
	var err error

	if includeInactive {
		membersList, total, err = h.svc.ListMembersIncludingInactive(c.Context(), tenantID, schoolID, role, offset, limit, search)
	} else {
		membersList, total, err = h.svc.ListMembers(c.Context(), tenantID, schoolID, role, offset, limit, search)
	}
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListResponse{
		Items: membersList,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ToggleActive handles PATCH /api/v1/members/:user_id/active
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
		"message": "member status updated",
	})
}

// GetByID handles GET /api/v1/members/:user_id
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

	member, err := h.svc.GetMemberByID(c.Context(), userID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(member)
}

// Update handles PUT /api/v1/members/:user_id
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

	var payload UpdateMemberPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UpdateMember(c.Context(), userID, tenantID, schoolID, payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "member updated",
	})
}

// Delete handles DELETE /api/v1/members/:user_id/:role
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var payload struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "malformed request body",
		})
	}

	payload.Role = strings.TrimSpace(payload.Role)
	if payload.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "user_id is required",
		})
	}
	if payload.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "role is required (TEACHER, NURSE, FINANCE, or SCHOOL_ADMIN)",
		})
	}

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	if err := h.svc.Delete(c.Context(), tenantID, schoolID, payload.UserID, payload.Role); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "member deleted",
	})
}

// Module is an fx-compatible module for the members domain.
var Module = fx.Module("members",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(Repository)),
			fx.As(new(ServiceRepository)),
		),
		NewService,
		NewHandler,
	),
)
