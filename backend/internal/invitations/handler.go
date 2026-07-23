package invitations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/fx"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/middleware"
)

// importServiceAdapter is the subset of imports.Service that the handler uses.
type importServiceAdapter interface {
	CreateJob(ctx context.Context, req imports.CreateJobRequest) (*imports.CreateJobResponse, error)
}

// Handler exposes invitation HTTP endpoints.
type Handler struct {
	svc    *Service
	impSvc importServiceAdapter
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetImportService sets the import service reference (called during DI wiring).
func (h *Handler) SetImportService(impSvc importServiceAdapter) {
	h.impSvc = impSvc
}

// RegisterRoutes mounts invitation routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Bulk invitation endpoint (scoped under /staff for semantic clarity)
	router.Post("/api/v1/staff/invite", middleware.RequireAuth, h.BulkInvite)

	// Invitation list/management endpoints
	invitations := router.Group("/api/v1/invitations")
	invitations.Get("/", middleware.RequireAuth, h.ListInvitations)
	invitations.Get("/count", middleware.RequireAuth, h.CountInvitations)
	invitations.Patch("/:id/revoke", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.RevokeInvitation)
}

// ─── Error response helper ─────────────────────────────────────────────────

type errorResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string, fieldErrors map[string][]string) error {
	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
		Errors:  fieldErrors,
	})
}

// ─── Bulk Invite ───────────────────────────────────────────────────────────

// BulkInvite handles POST /api/v1/staff/invite.
// Accepts a role and an array of email rows, creates an import job, processes
// invites asynchronously via Asynq workers, and returns immediately.
func (h *Handler) BulkInvite(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "active school not set", nil)
	}
	userID := c.Locals("user_id").(string)

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "invalid tenant", nil)
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "invalid school", nil)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "invalid user", nil)
	}

	var body BulkInviteRequest
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	// Validate role
	body.Role = strings.ToUpper(strings.TrimSpace(body.Role))
	if err := validateBulkInviteRole(body.Role); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input",
			fmt.Sprintf("invalid role %q — must be one of SCHOOL_ADMIN, TEACHER, NURSE, FINANCE", body.Role), nil)
	}

	// Validate at least one row
	if len(body.Rows) == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "rows array must not be empty",
			map[string][]string{"rows": {"At least one invitation is required"}})
	}

	// Validate row count limit (before CreateJob, before any DB writes)
	if len(body.Rows) > imports.MaxImportRows {
		return writeError(c, fiber.StatusBadRequest, "import_row_limit_exceeded",
			fmt.Sprintf("Invite list contains %d rows; the maximum is %d. Please split into smaller batches.",
				len(body.Rows), imports.MaxImportRows), nil)
	}

	// Resolve Stytch org ID for the tenant (needed by the importer at runtime)
	stytchOrgID, err := h.svc.GetStytchOrgID(c.Context(), tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// Build raw rows for the import engine
	rawRows := make([]json.RawMessage, len(body.Rows))
	for i, row := range body.Rows {
		data, _ := json.Marshal(row)
		rawRows[i] = json.RawMessage(data)
	}

	// Build metadata with the role, invited_by, import_job_id will be set
	// after job creation, and stytch_org_id for the importer
	meta := map[string]string{
		"role":          body.Role,
		"invited_by":    userID,
		"stytch_org_id": stytchOrgID,
		// import_job_id will be set below after job creation
	}
	metaJSON, _ := json.Marshal(meta)

	// Create the import job via the engine
	req := imports.CreateJobRequest{
		TenantID:  tenantUUID,
		SchoolID:  schoolUUID,
		JobType:   imports.ImportJobTypeStaffInvite,
		CreatedBy: userUUID,
		Role:      &body.Role,
		Rows:      rawRows,
		Metadata:  metaJSON,
	}

	resp, err := h.impSvc.CreateJob(c.Context(), req)
	if err != nil {
		if errors.Is(err, imports.ErrDuplicateJob) {
			return writeError(c, fiber.StatusConflict, "duplicate_import",
				"A job with this idempotency key already exists.", nil)
		}
		var inProgressErr *imports.ImportInProgressError
		if errors.As(err, &inProgressErr) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":          "import_already_in_progress",
				"message":       "An invitation job is already in progress for this school. Please wait for it to complete or cancel it.",
				"active_job_id": inProgressErr.ActiveJobID.String(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	// 201 for new job
	return c.Status(fiber.StatusCreated).JSON(BulkInviteResponse{
		JobID:        resp.JobID.String(),
		TotalRecords: resp.TotalRecords,
		TotalChunks:  resp.TotalChunks,
		Status:       string(resp.Status),
		IsReplay:     resp.IsReplay,
	})
}

// ─── List Invitations ──────────────────────────────────────────────────────

// ListInvitations handles GET /api/v1/invitations
func (h *Handler) ListInvitations(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := strings.TrimSpace(c.Query("search", ""))
	email := strings.TrimSpace(c.Query("email", ""))
	status := strings.TrimSpace(c.Query("status", ""))
	role := strings.TrimSpace(c.Query("role", ""))
	expired := strings.ToLower(c.Query("expired", "false")) == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "active school not set", nil)
	}

	invitations, total, err := h.svc.ListInvitations(c.Context(), tenantID, schoolID, ListInvitationsFilter{
		Search:  search,
		Email:   email,
		Status:  status,
		Role:    role,
		Expired: expired,
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListInvitationsResponse{
		Items: invitations,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ─── Count Invitations ─────────────────────────────────────────────────────

// CountInvitations handles GET /api/v1/invitations/count?role=SCHOOL_ADMIN
func (h *Handler) CountInvitations(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	role := strings.TrimSpace(c.Query("role", ""))

	if role == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "role query parameter is required", nil)
	}

	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "active school not set", nil)
	}

	total, err := h.svc.CountInvitations(c.Context(), tenantID, schoolID, role)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(CountInvitationsResponse{Total: total})
}

// RevokeInvitation handles PATCH /api/v1/invitations/:id/revoke
func (h *Handler) RevokeInvitation(c *fiber.Ctx) error {
	schoolID := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "active school not set", nil)
	}

	id := c.Params("id")
	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "invitation id is required", nil)
	}

	if err := h.svc.RevokeInvitation(c.Context(), id, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Module is an fx-compatible module for the invitations domain.
// Provides all dependencies including the StaffInviteImporter.
var Module = fx.Module("invitations",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		NewStaffInviteImporter,
	),
	// Wire import service adapter into the handler
	fx.Invoke(func(h *Handler, impSvc *imports.Service) {
		h.SetImportService(impSvc)
	}),
	// Wire Stytch adapter into the StaffInviteImporter
	fx.Invoke(func(sii *StaffInviteImporter, idp auth.IdentityProvider) {
		sii.SetStytchAdapter(idp)
	}),
	// Wire backend URL into the StaffInviteImporter (for Stytch invite redirect URL)
	fx.Invoke(func(sii *StaffInviteImporter, cfg config.Config) {
		sii.SetBackendURL(cfg.BackendURL)
	}),
	// Register the StaffInvite importer in the global registry
	fx.Invoke(registerStaffInviteImporter),
	// Provide the ParentInviteImporter
	fx.Provide(NewParentInviteImporter),
	// Wire Stytch adapter into the ParentInviteImporter
	fx.Invoke(func(pii *ParentInviteImporter, idp auth.IdentityProvider) {
		pii.SetStytchAdapter(idp)
	}),
	// Wire backend URL into the ParentInviteImporter (for Stytch invite redirect URL)
	fx.Invoke(func(pii *ParentInviteImporter, cfg config.Config) {
		pii.SetBackendURL(cfg.BackendURL)
	}),
	// Register the ParentInvite importer in the global registry
	fx.Invoke(registerParentInviteImporter),
)

// registerStaffInviteImporter registers the StaffInvite Importer.
func registerStaffInviteImporter(sii *StaffInviteImporter) {
	imports.RegisterImporter(sii)
}

// registerParentInviteImporter registers the ParentInvite importer.
func registerParentInviteImporter(pii *ParentInviteImporter) {
	imports.RegisterImporter(pii)
}
