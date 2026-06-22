package members

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/auth"
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

// RegisterRoutes mounts member and invitation routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	members := router.Group("/api/v1/members")
	members.Get("/", h.requireAuth, h.List)
	members.Post("/invite", h.requireAuth, h.BulkInvite)

	invitations := router.Group("/api/v1/invitations")
	invitations.Get("/", h.requireAuth, h.ListInvitations)
	invitations.Post("/", h.requireAuth, h.CreateInvitations)
}

// ─── Auth middleware ───────────────────────────────────────────────────────

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

// resolveActiveSchool gets the user's active school ID from their session.
func (h *Handler) resolveActiveSchool(c *fiber.Ctx, tenantID, userID string) (string, error) {
	schoolID, err := h.repo.GetActiveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("resolve active school: %w", err)
	}
	return schoolID, nil
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/members?role=TEACHER
//
// @Summary      List members by role
// @Description  Returns paginated members (users with active memberships) filtered by role.
// @Tags         Members
// @Produce      json
// @Param        role      query  string  true   "Role filter (TEACHER or NURSE)"
// @Param        page      query  int     false  "Page number (1-indexed)"
// @Param        per_page  query  int     false  "Items per page (max 100)"
// @Param        search    query  string  false  "Search by name or email"
// @Success      200  {object}  ListResponse
// @Failure      400  {object}  ErrorBody  "Invalid input"
// @Failure      401  {object}  ErrorBody  "Unauthorized"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/v1/members [get]
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	role := strings.TrimSpace(c.Query("role", ""))
	if role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "role query parameter is required (TEACHER, NURSE, or FINANCE)",
		})
	}
	if role != "TEACHER" && role != "NURSE" && role != "FINANCE" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "role must be TEACHER, NURSE, or FINANCE",
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
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve active school: " + err.Error(),
		})
	}

	members, total, err := h.svc.ListMembers(c.Context(), tenantID, schoolID, role, offset, perPage, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(ListResponse{
		Members: members,
		Total:   total,
	})
}

// BulkInvite handles POST /api/v1/members/invite
//
// @Summary      Bulk invite members
// @Description  Sends invitation emails to multiple people to join the school with a given role.
// @Tags         Members
// @Accept       json
// @Produce      json
// @Param        body  body  BulkInviteRequest  true  "Bulk invite payload"
// @Success      200  {object}  BulkInviteResponse
// @Failure      400  {object}  ErrorBody  "Invalid input"
// @Failure      401  {object}  ErrorBody  "Unauthorized"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/v1/members/invite [post]
func (h *Handler) BulkInvite(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var req BulkInviteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	if req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "role is required",
		})
	}
	if req.Role != "TEACHER" && req.Role != "NURSE" && req.Role != "FINANCE" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "role must be TEACHER, NURSE, or FINANCE",
		})
	}

	if len(req.Invites) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "at least one invite is required",
		})
	}

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve active school: " + err.Error(),
		})
	}

	result, err := h.svc.BulkInvite(c.Context(), tenantID, schoolID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(result)
}

// ─── Invitation Handlers ────────────────────────────────────────────────────

// ListInvitations handles GET /api/v1/invitations
//
// @Summary      List invitations
// @Description  Returns paginated invitations with optional filters.
// @Tags         Invitations
// @Produce      json
// @Param        search    query  string  false  "Search by name"
// @Param        email     query  string  false  "Filter by email"
// @Param        status    query  string  false  "Filter by status (pending, accepted, expired, revoked)"
// @Param        role      query  string  false  "Filter by role (TEACHER, SCHOOL_ADMIN, NURSE, FINANCE)"
// @Param        expired   query  bool    false  "Include expired invitations (default: false)"
// @Param        page      query  int     false  "Page number (1-indexed)"
// @Param        per_page  query  int     false  "Items per page (max 100)"
// @Success      200  {object}  ListInvitationsResponse
// @Failure      400  {object}  ErrorBody  "Invalid input"
// @Failure      401  {object}  ErrorBody  "Unauthorized"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/v1/invitations [get]
func (h *Handler) ListInvitations(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

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

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve active school: " + err.Error(),
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
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(ListInvitationsResponse{
		Invitations: invitations,
		Total:       total,
	})
}

// CreateInvitations handles POST /api/v1/invitations
//
// @Summary      Create invitations
// @Description  Creates new invitation records. This is the new endpoint that accepts per-invite roles.
// @Tags         Invitations
// @Accept       json
// @Produce      json
// @Param        body  body  CreateInvitationsRequest  true  "Invitations payload"
// @Success      200  {object}  BulkInviteResponse
// @Failure      400  {object}  ErrorBody  "Invalid input"
// @Failure      401  {object}  ErrorBody  "Unauthorized"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/v1/invitations [post]
func (h *Handler) CreateInvitations(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var req CreateInvitationsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	if len(req.Invites) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "at least one invite is required",
		})
	}

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve active school: " + err.Error(),
		})
	}

	result, err := h.svc.CreateInvitations(c.Context(), tenantID, schoolID, userID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(result)
}

// Module is an fx-compatible module for the members domain.
var Module = fx.Module("members",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
