package students

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"somotracker/backend/internal/academicyears"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/middleware"
	"somotracker/backend/internal/xerrors"
)

// importServiceAdapter is the subset of imports.Service that the handler uses.
type importServiceAdapter interface {
	CreateJob(ctx context.Context, req imports.CreateJobRequest) (*imports.CreateJobResponse, error)
}

// BehaviorNotesProvider is a function-based adapter that the students handler
// uses to fetch behavior notes for the student detail page. It is set via
// SetBehaviorNotesProvider during fx wiring in providers.go.
type BehaviorNotesProvider func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error)

// AttendanceSummaryProvider is a function-based adapter that the students handler
// uses to fetch attendance term summaries for the student detail page. It is set
// via SetAttendanceProvider during fx wiring in providers.go.
type AttendanceSummaryProvider func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error)

// Handler exposes student HTTP endpoints.
type Handler struct {
	svc              *Service
	impSvc           importServiceAdapter
	academicYearsSvc academicyears.AcademicYearTermResolver
	behaviorNotesFn  BehaviorNotesProvider
	attendanceFn     AttendanceSummaryProvider
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
func (h *Handler) SetAcademicYearsService(aySvc academicyears.AcademicYearTermResolver) {
	h.academicYearsSvc = aySvc
}

// SetBehaviorNotesProvider sets the function that fetches behavior notes
// for the student detail page. Wired from providers.go.
func (h *Handler) SetBehaviorNotesProvider(fn BehaviorNotesProvider) {
	h.behaviorNotesFn = fn
}

// SetAttendanceProvider sets the function that fetches attendance term summaries
// for the student detail page. Wired from providers.go.
func (h *Handler) SetAttendanceProvider(fn AttendanceSummaryProvider) {
	h.attendanceFn = fn
}

// RegisterRoutes mounts student routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	students := router.Group("/api/v1/students")
	students.Get("/list", middleware.RequireAuth, h.List)
	students.Post("/", middleware.RequireAuth, h.Create)
	students.Get("/:id", middleware.RequireAuth, h.GetDetail)
	students.Put("/:id", middleware.RequireAuth, h.Update)
	students.Delete("/", middleware.RequireAuth, h.Delete)

	// Enrollments
	students.Post("/enrollments", middleware.RequireAuth, h.CreateBatchEnrollments)
	students.Post("/:id/enrollments", middleware.RequireAuth, h.CreateEnrollment)
	students.Get("/:id/enrollments", middleware.RequireAuth, h.ListEnrollments)

	// Bulk import (with request body size limit scoped to this route only)
	students.Post("/import", middleware.RequireAuth, bodySizeLimit, h.BulkImport)

	// Duplicate checking (proactive check used by frontend before submit)
	students.Post("/check-duplicates", middleware.RequireAuth, h.CheckDuplicates)
}

// bodySizeLimit is a per-route middleware that rejects requests whose
// Content-Length exceeds maxImportBodyBytes. This is scoped to the import
// endpoint and does NOT apply a codebase-wide body limit.
//
// Assumption: 5000 rows × ~2KB/row ≈ 10MB. With 50% margin the cap is
// 15MB (imports.maxImportBodyBytes). If the Content-Length header is
// missing the body is parsed normally (Fiber clamps internally at 4MB by
// default, but that default may change in future Fiber versions).
func bodySizeLimit(c *fiber.Ctx) error {
	if cl := c.Get("Content-Length"); cl != "" {
		if size, err := strconv.Atoi(cl); err == nil && size > imports.MaxImportBodyBytes() {
			return middleware.HTTPError(c, xerrors.New("request_too_large", fiber.StatusRequestEntityTooLarge, fmt.Sprintf("Request body is too large (%d bytes). The maximum is %d bytes for student import. Please reduce the number of rows.",
				size, imports.MaxImportBodyBytes())))
		}
	}
	return c.Next()
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
		return middleware.HTTPError(c, xerrors.InvalidInput("active school not set"))
	}
	userID := c.Locals("user_id").(string)

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return middleware.HTTPError(c, xerrors.InvalidInput("invalid tenant"))
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return middleware.HTTPError(c, xerrors.InvalidInput("invalid school"))
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return middleware.HTTPError(c, xerrors.InvalidInput("invalid user"))
	}

	var body ImportRequest
	if err := c.BodyParser(&body); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}

	// Validate at least one row
	if len(body.Rows) == 0 {
		return middleware.HTTPError(c, xerrors.WithFields(xerrors.InvalidInput("rows array must not be empty"), map[string][]string{"rows": {"At least one row is required"}}))
	}

	// Validate row count limit (before CreateJob, before any DB writes)
	if len(body.Rows) > imports.MaxImportRows {
		return middleware.HTTPError(c, xerrors.InvalidInput(fmt.Sprintf("Import contains %d rows; the maximum is %d. Please split into smaller files.",
			len(body.Rows), imports.MaxImportRows)))
	}

	// Resolve current active academic year and term server-side
	academicYearID, academicTermID, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("no active academic year"))
	}
	if academicTermID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("no active academic term"))
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
			return middleware.HTTPError(c, xerrors.AlreadyExists("a job with this idempotency key already exists"))
		}
		var inProgressErr *imports.ImportInProgressError
		if errors.As(err, &inProgressErr) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":          "import_already_in_progress",
				"message":       "An import job is already in progress for this school. Please wait for it to complete or cancel it.",
				"active_job_id": inProgressErr.ActiveJobID.String(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	// 200 for idempotent replay, 201 for new job
	httpStatus := fiber.StatusCreated
	if resp.IsReplay {
		httpStatus = fiber.StatusOK
	}

	return c.Status(httpStatus).JSON(ImportResponse{
		JobID:        resp.JobID.String(),
		TotalRecords: resp.TotalRecords,
		TotalChunks:  resp.TotalChunks,
		Status:       string(resp.Status),
		IsReplay:     resp.IsReplay,
	})
}

// ─── Check Duplicates ─────────────────────────────────────────────────────

// CheckDuplicates handles POST /api/v1/students/check-duplicates.
// For each provided list of values, returns only those that already exist
// in cbc_students for the caller's tenant/school. All three fields are
// optional — omitted fields are not checked.
func (h *Handler) CheckDuplicates(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("active school not set"))
	}

	var body CheckDuplicatesRequest
	if err := c.BodyParser(&body); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}

	existingAdm, existingUPI, existingKNEC, err := h.svc.CheckDuplicates(
		c.Context(), tenantID, schoolID,
		body.AdmissionNumbers, body.UPINumbers, body.KNECAssessmentNumbers,
	)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(CheckDuplicatesResponse{
		ExistingAdmissionNumbers:      existingAdm,
		ExistingUPINumbers:            existingUPI,
		ExistingKNECAssessmentNumbers: existingKNEC,
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
		return middleware.HTTPError(c, xerrors.InvalidInput("active school not set"))
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
		return middleware.HTTPError(c, xerrors.InvalidInput("active school not set"))
	}

	var body CreateStudentsPayload
	if err := c.BodyParser(&body); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}

	if len(body.Students) == 0 {
		return middleware.HTTPError(c, xerrors.WithFields(xerrors.InvalidInput("students array must not be empty"), map[string][]string{"students": {"At least one student is required"}}))
	}

	// Validate all entries have required fields before creating any
	for _, s := range body.Students {
		if s.FullName == "" {
			return middleware.HTTPError(c, xerrors.WithFields(xerrors.InvalidInput("full_name is required for all students"), map[string][]string{"students": {}}))
		}
	}

	result, err := h.svc.CreateBatch(c.Context(), tenantID, schoolID, body.Students)
	if err != nil {
		if errors.Is(err, ErrDuplicateUPI) {
			return middleware.HTTPError(c, xerrors.WithFields(xerrors.AlreadyExists("a student with this UPI number already exists"), map[string][]string{"upi_number": {"This UPI number is already in use"}}))
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// ─── Get Detail ───────────────────────────────────────────────────────────

// GetDetail handles GET /api/v1/students/:id.
// Supports optional ?term_id for scoping behavior notes.
func (h *Handler) GetDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	id := c.Params("id")

	if id == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("student id is required"))
	}

	detail, err := h.svc.GetDetail(c.Context(), id, tenantID, schoolID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return middleware.HTTPError(c, xerrors.NotFound("student not found"))
		}
		return middleware.HTTPError(c, err)
	}

	// Fetch behavior notes if the provider is wired in
	if h.behaviorNotesFn != nil {
		termID := c.Query("term_id")
		if termID != "" {
			notes, err := h.behaviorNotesFn(c.Context(), tenantID, schoolID, id, termID)
			if err == nil && notes != nil {
				detail.Behavior = notes
			}
		}
	}

	// Fetch attendance summaries if the provider is wired in
	if h.attendanceFn != nil {
		termID := c.Query("term_id")
		if termID != "" {
			attendance, err := h.attendanceFn(c.Context(), tenantID, schoolID, id, termID)
			if err == nil && attendance != nil {
				detail.Attendance = attendance
			}
		}
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
		return middleware.HTTPError(c, xerrors.InvalidInput("student id is required"))
	}

	var body UpdateStudentPayload
	if err := c.BodyParser(&body); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}

	if err := h.svc.Update(c.Context(), id, tenantID, schoolID, body); err != nil {
		if errors.Is(err, ErrDuplicateUPI) {
			return middleware.HTTPError(c, xerrors.WithFields(xerrors.AlreadyExists("a student with this UPI number already exists"), map[string][]string{"upi_number": {"This UPI number is already in use"}}))
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ─── Create Enrollment ────────────────────────────────────────────────────

// CreateEnrollment handles POST /api/v1/students/:id/enrollments.
// academic_term_id is resolved server-side from the current active term.
func (h *Handler) CreateEnrollment(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	studentID := c.Params("id")

	if studentID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("student id is required"))
	}

	var body CreateEnrollmentPayload
	if err := c.BodyParser(&body); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}

	if body.ClassID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("class_id is required"))
	}

	// Resolve current academic term server-side
	academicYearID, academicTermID, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("no active academic year"))
	}
	if academicTermID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("no active academic term"))
	}

	// Set the resolved academic term ID on the body before passing to service
	body.AcademicTermID = academicTermID

	enrollment, err := h.svc.CreateEnrollment(c.Context(), studentID, tenantID, schoolID, body)
	if err != nil {
		if errors.Is(err, ErrDuplicateEnroll) {
			return middleware.HTTPError(c, xerrors.AlreadyExists("this student is already enrolled in this term"))
		}
		if errors.Is(err, ErrNotFound) {
			return middleware.HTTPError(c, xerrors.NotFound("student not found"))
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateEnrollmentResponse{ID: enrollment.ID})
}

// ─── Batch Enrollments ───────────────────────────────────────────────────

// CreateBatchEnrollments handles POST /api/v1/students/enrollments.
// Accepts a list of { student_id, class_id } pairs. Academic term is resolved
// server-side from the current active term. Status defaults to ACTIVE.
func (h *Handler) CreateBatchEnrollments(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("active school not set"))
	}

	var body BatchEnrollRequest
	if err := c.BodyParser(&body); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}

	if len(body.Enrollments) == 0 {
		return middleware.HTTPError(c, xerrors.WithFields(xerrors.InvalidInput("enrollments array must not be empty"), map[string][]string{"enrollments": {"At least one enrollment is required"}}))
	}

	// Validate each item has required fields
	for i, item := range body.Enrollments {
		if item.StudentID == "" {
			studentIDKey := fmt.Sprintf("enrollments[%d].student_id", i)
			fieldErrors := map[string][]string{studentIDKey: {"This field is required"}}
			return middleware.HTTPError(c, xerrors.WithFields(xerrors.InvalidInput("student_id is required for all enrollments"), fieldErrors))
		}
		if item.ClassID == "" {
			classIDKey := fmt.Sprintf("enrollments[%d].class_id", i)
			fieldErrors := map[string][]string{classIDKey: {"This field is required"}}
			return middleware.HTTPError(c, xerrors.WithFields(xerrors.InvalidInput("class_id is required for all enrollments"), fieldErrors))
		}
	}

	// Resolve current academic year and term server-side
	academicYearID, academicTermID, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("no active academic year"))
	}
	if academicTermID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("no active academic term"))
	}

	result, err := h.svc.CreateBatchEnrollments(c.Context(), tenantID, schoolID, academicTermID, body.Enrollments)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// Delete handles DELETE /api/v1/students/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("malformed request body"))
	}
	if payload.ID == "" {
		return middleware.HTTPError(c, xerrors.InvalidInput("student id is required"))
	}

	if err := h.svc.Delete(c.Context(), payload.ID, tenantID, schoolID); err != nil {
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
		return middleware.HTTPError(c, xerrors.InvalidInput("student id is required"))
	}

	enrollments, err := h.svc.ListEnrollments(c.Context(), studentID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListEnrollmentsResponse{
		Items: enrollments,
		Total: len(enrollments),
		Page:  1,
		Limit: len(enrollments),
	})
}
