package invitations

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// Handler exposes invitation HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts invitation routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	invitations := router.Group("/api/v1/invitations")
	invitations.Get("/", middleware.RequireAuth, h.ListInvitations)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// ListInvitations handles GET /api/v1/invitations
func (h *Handler) ListInvitations(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	search := strings.TrimSpace(c.Query("search", ""))
	email := strings.TrimSpace(c.Query("email", ""))
	status := strings.TrimSpace(c.Query("status", ""))
	role := strings.TrimSpace(c.Query("role", ""))
	expired := strings.ToLower(c.Query("expired", "false")) == "true"

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

	invitations, total, err := h.svc.ListInvitations(c.Context(), tenantID, schoolID, ListInvitationsFilter{
		Search:  search,
		Email:   email,
		Status:  status,
		Role:    role,
		Expired: expired,
		Offset:  offset,
		Limit:   perPage,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListInvitationsResponse{
		Invitations: invitations,
		Total:       total,
	})
}

// Module is an fx-compatible module for the invitations domain.
var Module = fx.Module("invitations",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
