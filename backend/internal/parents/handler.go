package parents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/fx"

	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/middleware"
)

// importServiceAdapter is the subset of imports.Service that the handler uses.
type importServiceAdapter interface {
	CreateJob(ctx context.Context, req imports.CreateJobRequest) (*imports.CreateJobResponse, error)
}

// ============================================================================
// Handler
// ============================================================================

// Handler exposes parent HTTP endpoints.
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

// RegisterRoutes mounts parent routes on the given router.
// Note: static routes (/me) must be registered before parameterised routes (/:id)
// to prevent path parameter matching.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	parents := router.Group("/api/v1/parents")
	parents.Post("/", middleware.RequireAuth, h.Create)
	parents.Get("/", middleware.RequireAuth, h.List)
	parents.Get("/me", middleware.RequireAuth, h.GetMyProfile)
	parents.Post("/invite", middleware.RequireAuth, h.BulkInvite)
	parents.Post("/import", middleware.RequireAuth, h.BulkImport)
	parents.Get("/:id", middleware.RequireAuth, h.GetDetail)
	parents.Put("/:id", middleware.RequireAuth, h.Update)
	parents.Delete("/:id", middleware.RequireAuth, h.Delete)
	parents.Post("/:parent_id/students", middleware.RequireAuth, h.LinkStudent)
	parents.Delete("/:parent_id/students/:student_id", middleware.RequireAuth, h.UnlinkStudent)
}

// ============================================================================
// Error response helper
// ============================================================================

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

// ============================================================================
// CREATE
// ============================================================================

// Create handles POST /api/v1/parents.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var body CreateParentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.Email == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "email is required", nil)
	}
	if body.FullName == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "full_name is required", nil)
	}
	if body.PhoneNumber == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "phone_number is required", nil)
	}

	parent, err := h.svc.Create(c.Context(), tenantID, body)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return writeError(c, fiber.StatusConflict, "already_exists",
				"A parent profile for this email already exists in this school.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateParentResponse{ID: parent.ID})
}

// ============================================================================
// BULK INVITE
// ============================================================================

// BulkInvite handles POST /api/v1/parents/invite.
// Accepts an array of email rows, creates a PARENT_INVITE import job, processes
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

	// Validate at least one row
	if len(body.Rows) == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "rows array must not be empty",
			map[string][]string{"rows": {"At least one invitation is required"}})
	}

	// Validate row count limit
	if len(body.Rows) > imports.MaxImportRows {
		return writeError(c, fiber.StatusBadRequest, "import_row_limit_exceeded",
			fmt.Sprintf("Invite list contains %d rows; the maximum is %d. Please split into smaller batches.",
				len(body.Rows), imports.MaxImportRows), nil)
	}

	// Resolve Stytch org ID for the tenant
	stytchOrgID, err := h.svc.GetStytchOrgID(c.Context(), tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// Build raw rows for the import engine
	role := "PARENT"
	rawRows := make([]json.RawMessage, len(body.Rows))
	for i, row := range body.Rows {
		data, _ := json.Marshal(row)
		rawRows[i] = json.RawMessage(data)
	}

	// Build metadata
	meta := map[string]string{
		"role":          role,
		"invited_by":    userID,
		"stytch_org_id": stytchOrgID,
	}
	metaJSON, _ := json.Marshal(meta)

	// Create the import job via the engine
	req := imports.CreateJobRequest{
		TenantID:  tenantUUID,
		SchoolID:  schoolUUID,
		JobType:   imports.ImportJobTypeParentInvite,
		CreatedBy: userUUID,
		Role:      &role,
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

	return c.Status(fiber.StatusCreated).JSON(BulkInviteResponse{
		JobID:        resp.JobID.String(),
		TotalRecords: resp.TotalRecords,
		TotalChunks:  resp.TotalChunks,
		Status:       string(resp.Status),
		IsReplay:     resp.IsReplay,
	})
}

// ============================================================================
// BULK IMPORT (legacy — deprecated)
// ============================================================================

// BulkImport handles POST /api/v1/parents/import (kept for backward compatibility).
func (h *Handler) BulkImport(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"code":    "not_implemented",
		"message": "Bulk import for parents is not yet implemented. Use /api/v1/parents/invite instead.",
	})
}

// ============================================================================
// LIST
// ============================================================================

// List handles GET /api/v1/parents.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := strings.TrimSpace(c.Query("search"))
	studentID := strings.TrimSpace(c.Query("student_id"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	filter := ListFilter{
		TenantID:  tenantID,
		Search:    search,
		StudentID: studentID,
		Page:      page,
		Limit:     limit,
	}

	// Parse multi-value query params: ?education_level=Early_Years&education_level=Upper_Primary
	if parsedURL, err := url.Parse(c.OriginalURL()); err == nil {
		if vals := parsedURL.Query()["education_level"]; len(vals) > 0 {
			filter.EducationLevels = vals
		}
		if vals := parsedURL.Query()["grade_level"]; len(vals) > 0 {
			filter.GradeLevels = vals
		}
	}

	parents, total, err := h.svc.List(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListParentsResponse{
		Items: parents,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ============================================================================
// GET DETAIL
// ============================================================================

// GetDetail handles GET /api/v1/parents/:id.
func (h *Handler) GetDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent id is required", nil)
	}

	detail, err := h.svc.GetDetail(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ParentDetailResponse{Data: *detail})
}

// GetMyProfile handles GET /api/v1/parents/me.
// Returns the authenticated parent's profile with linked children.
func (h *Handler) GetMyProfile(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	parent, err := h.svc.GetByUserID(c.Context(), userID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	detail, err := h.svc.GetDetail(c.Context(), parent.ID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ParentDetailResponse{Data: *detail})
}

// ============================================================================
// UPDATE
// ============================================================================

// Update handles PUT /api/v1/parents/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent id is required", nil)
	}

	var body UpdateParentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.PhoneNumber == nil && body.IsActive == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input",
			"at least one of phone_number or is_active must be provided", nil)
	}

	if err := h.svc.Update(c.Context(), id, tenantID, body); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// DELETE
// ============================================================================

// Delete handles DELETE /api/v1/parents/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent id is required", nil)
	}

	if err := h.svc.Delete(c.Context(), id, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// LINK STUDENT
// ============================================================================

// LinkStudent handles POST /api/v1/parents/:parent_id/students.
func (h *Handler) LinkStudent(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	parentID := c.Params("parent_id")

	if parentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent_id is required", nil)
	}

	var body LinkStudentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.StudentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student_id is required", nil)
	}

	if err := h.svc.LinkStudent(c.Context(), parentID, tenantID, body); err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			return writeError(c, fiber.StatusNotFound, "student_not_found",
				"The specified student was not found in this school.", nil)
		}
		if errors.Is(err, ErrDuplicateLink) {
			return writeError(c, fiber.StatusConflict, "duplicate_link",
				"This student is already linked to this parent.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// UNLINK STUDENT
// ============================================================================

// UnlinkStudent handles DELETE /api/v1/parents/:parent_id/students/:student_id.
func (h *Handler) UnlinkStudent(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	parentID := c.Params("parent_id")
	studentID := c.Params("student_id")

	if parentID == "" || studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent_id and student_id are required", nil)
	}

	if err := h.svc.UnlinkStudent(c.Context(), parentID, studentID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// fx Module
// ============================================================================

// Module is an fx-compatible module for the parents domain.
var Module = fx.Module("parents",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(Repository)),
			fx.As(new(StudentResolver)),
		),
		NewService,
		NewHandler,
	),
	// Wire import service adapter into the handler
	fx.Invoke(func(h *Handler, impSvc *imports.Service) {
		h.SetImportService(impSvc)
	}),
)
