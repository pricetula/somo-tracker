package students

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/middleware"
)

// importServiceAdapter is the subset of imports.Service that the handler uses.
type importServiceAdapter interface {
	CreateJob(ctx context.Context, req imports.CreateJobRequest) (*imports.CreateJobResponse, error)
}

// academicYearsAdapter is the subset of academicyears.Service that the handler uses.
type academicYearsAdapter interface {
	GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error)
	GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error)
}

// Handler exposes student HTTP endpoints.
type Handler struct {
	svc              *Service
	impSvc           importServiceAdapter
	academicYearsSvc academicYearsAdapter
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetImportService sets the import service reference.
func (h *Handler) SetImportService(impSvc importServiceAdapter) {
	h.impSvc = impSvc
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicYearsAdapter) {
	h.academicYearsSvc = aySvc
}

// RegisterRoutes mounts student routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	students := router.Group("/api/v1/students")
	students.Get("/list", middleware.RequireAuth, h.List)
	students.Post("/", middleware.RequireAuth, h.Create)
	students.Get("/:id", middleware.RequireAuth, h.GetDetail)
	students.Put("/:id", middleware.RequireAuth, h.Update)

	// Enrollments (nested under students)
	students.Post("/:id/enrollments", middleware.RequireAuth, h.CreateEnrollment)
	students.Get("/:id/enrollments", middleware.RequireAuth, h.ListEnrollments)

	// Bulk import
	students.Post("/import", middleware.RequireAuth, h.BulkImport)
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

// ─── Bulk Import ───────────────────────────────────────────────────────────

// BulkImport handles POST /api/v1/students/import.
func (h *Handler) BulkImport(c *fiber.Ctx) error {
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

	var body ImportRequest
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	// Validate at least one row
	if len(body.Rows) == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "rows array must not be empty",
			map[string][]string{"rows": {"At least one row is required"}})
	}

	// Resolve current active academic year and term server-side
	academicYearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return writeError(c, fiber.StatusBadRequest, "no_active_academic_year",
			"No current academic year is set for this school. Please set one before importing.", nil)
	}

	academicTermID, err := h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicTermID == "" {
		return writeError(c, fiber.StatusBadRequest, "no_active_academic_term",
			"No current academic term is active. Please set one before importing.", nil)
	}

	// Build metadata with academic context
	meta := map[string]string{
		"academic_term_id": academicTermID,
		"academic_year_id": academicYearID,
	}
	metaJSON, _ := json.Marshal(meta)

	// Build raw rows for the import engine
	rawRows := make([]json.RawMessage, len(body.Rows))
	for i, row := range body.Rows {
		data, _ := json.Marshal(row)
		rawRows[i] = json.RawMessage(data)
	}

	// Create the import job via the engine
	req := imports.CreateJobRequest{
		TenantID:       tenantUUID,
		SchoolID:       schoolUUID,
		JobType:        imports.ImportJobTypeStudentImport,
		CreatedBy:      userUUID,
		Rows:           rawRows,
		IDempotencyKey: body.IDempotencyKey,
		Metadata:       metaJSON,
	}

	resp, err := h.impSvc.CreateJob(c.Context(), req)
	if err != nil {
		if errors.Is(err, imports.ErrDuplicateJob) {
			return writeError(c, fiber.StatusConflict, "duplicate_import",
				"A job with this idempotency key already exists.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(ImportResponse{
		JobID:        resp.JobID.String(),
		TotalRecords: resp.TotalRecords,
		TotalChunks:  resp.TotalChunks,
		Status:       string(resp.Status),
	})
}

// ─── List ─────────────────────────────────────────────────────────────────

// List handles GET /api/v1/students/list.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Multi-value filter params
	filter := ListFilter{
		TenantID:         tenantID,
		SchoolID:         schoolID,
		Page:             page,
		Limit:            limit,
		Search:           c.Query("search"),
		ClassID:          c.Query("class_id"),
		Gender:           c.Query("gender"),
		EnrollmentStatus: c.Query("enrollment_status"),
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

	result, err := h.svc.ListStudents(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ─── Create ───────────────────────────────────────────────────────────────

// Create handles POST /api/v1/students.
// Accepts a batch payload: { "students": [{ ... }, ...] } and creates all
// students in a single transaction. Returns the array of created IDs.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "active school not set", nil)
	}

	var body CreateStudentsPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if len(body.Students) == 0 {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "students array must not be empty",
			map[string][]string{"students": {"At least one student is required"}})
	}

	// Validate all entries have required fields before creating any
	for i, s := range body.Students {
		if s.FullName == "" {
			return writeError(c, fiber.StatusBadRequest, "invalid_input",
				"full_name is required for all students",
				map[string][]string{
					"students": {},
				})
		}
		_ = i // used for potential future per-field error reporting
	}

	result, err := h.svc.CreateBatch(c.Context(), tenantID, schoolID, body.Students)
	if err != nil {
		if errors.Is(err, ErrDuplicateUPI) {
			return writeError(c, fiber.StatusConflict, "duplicate_upi",
				"A student with this UPI number already exists.",
				map[string][]string{"upi_number": {"This UPI number is already in use"}})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// ─── Get Detail ───────────────────────────────────────────────────────────

// GetDetail handles GET /api/v1/students/:id.
func (h *Handler) GetDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student id is required", nil)
	}

	detail, err := h.svc.GetDetail(c.Context(), id, tenantID, schoolID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    "not_found",
				"message": "Student not found",
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.JSON(StudentDetailResponse{Data: *detail})
}

// ─── Update ───────────────────────────────────────────────────────────────

// Update handles PUT /api/v1/students/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student id is required", nil)
	}

	var body UpdateStudentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if err := h.svc.Update(c.Context(), id, tenantID, schoolID, body); err != nil {
		if errors.Is(err, ErrDuplicateUPI) {
			return writeError(c, fiber.StatusConflict, "duplicate_upi",
				"A student with this UPI number already exists.",
				map[string][]string{"upi_number": {"This UPI number is already in use"}})
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ─── Create Enrollment ────────────────────────────────────────────────────

// CreateEnrollment handles POST /api/v1/students/:id/enrollments.
func (h *Handler) CreateEnrollment(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	studentID := c.Params("id")

	if studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student id is required", nil)
	}

	var body CreateEnrollmentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.AcademicTermID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "academic_term_id is required", nil)
	}
	if body.ClassID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "class_id is required", nil)
	}

	enrollment, err := h.svc.CreateEnrollment(c.Context(), studentID, tenantID, schoolID, body)
	if err != nil {
		if errors.Is(err, ErrDuplicateEnroll) {
			return writeError(c, fiber.StatusConflict, "duplicate_enrollment",
				"This student is already enrolled in this term.", nil)
		}
		if errors.Is(err, ErrNotFound) {
			return writeError(c, fiber.StatusNotFound, "not_found", "Student not found", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateEnrollmentResponse{ID: enrollment.ID})
}

// Delete handles DELETE /api/v1/students/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student id is required", nil)
	}

	if err := h.svc.Delete(c.Context(), id, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "student deleted",
	})
}

// ─── List Enrollments ─────────────────────────────────────────────────────

// ListEnrollments handles GET /api/v1/students/:id/enrollments.
func (h *Handler) ListEnrollments(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	studentID := c.Params("id")

	if studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student id is required", nil)
	}

	enrollments, err := h.svc.ListEnrollments(c.Context(), studentID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListEnrollmentsResponse{Items: enrollments})
}
