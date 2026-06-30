package members

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/middleware"
)

// Handler exposes member HTTP endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
	repo    Repository
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, authSvc *auth.Service, repo Repository) *Handler {
	return &Handler{svc: svc, authSvc: authSvc, repo: repo}
}

// RegisterRoutes mounts member routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	members := router.Group("/api/v1/members")
	members.Get("/", h.requireAuth, h.List)
}

// ─── Auth middleware ───────────────────────────────────────────────────────

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	token := c.Cookies("somo_sid")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "no session cookie found",
		})
	}

	session, err := h.authSvc.GetSession(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "invalid or expired session",
		})
	}

	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	return c.Next()
}

// resolveActiveSchool gets the user's active school ID from their session.
func (h *Handler) resolveActiveSchool(c *fiber.Ctx, tenantID, userID string) (string, error) {
	schoolID, err := h.repo.GetActiveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("members.Handler.resolveActiveSchool: %w", err)
	}
	return schoolID, nil
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/members?role=TEACHER
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

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

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to resolve active school",
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
