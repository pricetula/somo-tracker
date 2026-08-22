package cbcclasses

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// getQuerySlice returns all values for a given query parameter key.
// Handles repeated keys like ?grade_level=G7&grade_level=G8.
func getQuerySlice(c *fiber.Ctx, key string) []string {
	var values []string
	for _, v := range c.Request().URI().QueryArgs().PeekMulti(key) {
		values = append(values, string(v))
	}
	return values
}

// academicYearsAdapter is the subset of academicyears.Service that the handler uses.
type academicYearsAdapter interface {
	GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error)
	GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error)
}

// Handler exposes class HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc academicYearsAdapter
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicYearsAdapter) {
	h.academicYearsSvc = aySvc
}

// RegisterRoutes mounts class routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	classes := router.Group("/api/v1/classes")
	classes.Get("/", middleware.RequireAuth, h.List)
	classes.Get("/:id", middleware.RequireAuth, h.Get)
	classes.Post("/", middleware.RequireAuth, h.Create)
	classes.Put("/:id", middleware.RequireAuth, h.Update)
	classes.Delete("/", middleware.RequireAuth, h.BulkDelete)

	// Enrollment routes
	classes.Get("/:id/roster", middleware.RequireAuth, h.GetRoster)
	classes.Post("/:id/enroll", middleware.RequireAuth, h.BatchEnroll)
	classes.Post("/:id/unenroll/:studentId", middleware.RequireAuth, h.UnenrollStudent)
	classes.Get("/:id/available-students", middleware.RequireAuth, h.GetAvailableStudents)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/classes.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		var err error
		academicYearID, err = h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		// No current academic year configured — nothing to list
		if academicYearID == "" {
			return c.JSON(ClassListResult{Items: []Class{}, Total: 0, Page: 1, Limit: 50})
		}
	}

	academicTermID := c.Query("academic_term_id")
	if academicTermID == "" {
		var err error
		academicTermID, err = h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		// No current academic term configured — nothing to list
		if academicTermID == "" {
			return c.JSON(ClassListResult{Items: []Class{}, Total: 0, Page: 1, Limit: 50})
		}
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

	filter := ClassListFilter{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: academicYearID,
		AcademicTermID: academicTermID,
		Search:         c.Query("search"),
		Page:           page,
		Limit:          limit,
	}

	if gradeLevels := getQuerySlice(c, "grade_level"); len(gradeLevels) > 0 {
		filter.GradeLevels = gradeLevels
	}
	if streamIDs := getQuerySlice(c, "stream_id"); len(streamIDs) > 0 {
		filter.StreamIDs = streamIDs
	}

	result, err := h.svc.ListClasses(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// Get handles GET /api/v1/classes/:id.
func (h *Handler) Get(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	classID := c.Params("id")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "class id is required",
		})
	}

	class, err := h.svc.GetClass(c.Context(), classID, tenantID, schoolID)
	if err != nil {
		return mapClassError(c, err)
	}

	// Get student count from roster — scope to the provided term or current
	academicTermID := c.Query("academic_term_id")
	if academicTermID == "" {
		academicYearID, yearErr := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
		if yearErr == nil && academicYearID != "" {
			if tID, tErr := h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID); tErr == nil && tID != "" {
				rosterResult, rosterErr := h.svc.GetRoster(c.Context(), classID, tenantID, schoolID, tID, 1, 1, "")
				if rosterErr == nil {
					class.StudentCount = rosterResult.Total
				}
			}
		}
	} else {
		rosterResult, rosterErr := h.svc.GetRoster(c.Context(), classID, tenantID, schoolID, academicTermID, 1, 1, "")
		if rosterErr == nil {
			class.StudentCount = rosterResult.Total
		}
	}

	return c.JSON(class)
}

// Create handles POST /api/v1/classes.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	var payload CreateClassPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Validate required fields
	if payload.GradeLevel == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "grade_level is required",
			"errors":  map[string][]string{"grade_level": {"Grade level is required"}},
		})
	}
	if payload.StreamID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "stream_id is required",
			"errors":  map[string][]string{"stream_id": {"Stream is required"}},
		})
	}

	// Resolve current academic year and term
	yearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if yearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}

	termID, err := h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_TERM",
			"message": "No current academic term is active.",
		})
	}

	if payload.StudentIDs == nil {
		payload.StudentIDs = []string{}
	}

	params := CreateClassParams{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		AcademicTermID: termID,
		GradeLevel:     payload.GradeLevel,
		StreamID:       payload.StreamID,
		StudentIDs:     payload.StudentIDs,
	}

	class, err := h.svc.CreateClass(c.Context(), params)
	if err != nil {
		return mapClassError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(class)
}

// Update handles PUT /api/v1/classes/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	classID := c.Params("id")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "class id is required",
		})
	}

	var payload UpdateClassPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.GradeLevel == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "grade_level is required",
		})
	}
	if payload.StreamID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "stream_id is required",
		})
	}

	if payload.StudentIDs == nil {
		payload.StudentIDs = []string{}
	}

	// Resolve current academic term server-side
	academicYearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}

	academicTermID, err := h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicTermID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_TERM",
			"message": "No current academic term is active.",
		})
	}

	params := UpdateClassParams{
		ClassID:        classID,
		TenantID:       tenantID,
		SchoolID:       schoolID,
		GradeLevel:     payload.GradeLevel,
		StreamID:       payload.StreamID,
		AcademicTermID: academicTermID,
		StudentIDs:     payload.StudentIDs,
	}

	class, err := h.svc.UpdateClass(c.Context(), params)
	if err != nil {
		return mapClassError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(class)
}

// BulkDelete handles DELETE /api/v1/classes.
func (h *Handler) BulkDelete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	var payload BulkDeletePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if len(payload.ClassIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "class_ids is required",
		})
	}

	if len(payload.ClassIDs) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "LIMIT_EXCEEDED",
			"message": "max 100 class IDs per request",
		})
	}

	if err := h.svc.BulkDeleteClasses(c.Context(), payload.ClassIDs, tenantID, schoolID); err != nil {
		return mapClassError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ─── Get Roster ──────────────────────────────────────────────────────────

// GetRoster handles GET /api/v1/classes/:id/roster.
func (h *Handler) GetRoster(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	classID := c.Params("id")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "class id is required",
		})
	}

	academicYearID := c.Query("academic_year_id")
	academicTermID := c.Query("academic_term_id")

	// If neither provided, fall back to the current academic year/term
	if academicYearID == "" && academicTermID == "" {
		var err error
		academicYearID, err = h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicYearID == "" {
			return c.JSON(RosterListResult{Items: []RosterEntry{}, Total: 0, Page: 1, Limit: 50})
		}
	}

	// If year is provided but term is not, resolve current term for that year
	if academicTermID == "" && academicYearID != "" {
		var err error
		academicTermID, err = h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicTermID == "" {
			return c.JSON(RosterListResult{Items: []RosterEntry{}, Total: 0, Page: 1, Limit: 50})
		}
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

	search := c.Query("search")

	result, err := h.svc.GetRoster(c.Context(), classID, tenantID, schoolID, academicTermID, page, limit, search)
	if err != nil {
		return mapClassError(c, err)
	}

	return c.JSON(result)
}

// ─── Batch Enroll ─────────────────────────────────────────────────────────

// BatchEnroll handles POST /api/v1/classes/:id/enroll.
func (h *Handler) BatchEnroll(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	classID := c.Params("id")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "class id is required",
		})
	}

	var payload BatchEnrollPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Use the provided academic term or resolve the current one
	academicTermID := payload.AcademicTermID
	if academicTermID == "" {
		academicYearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicYearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is set for this school.",
			})
		}

		academicTermID, err = h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicTermID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	if len(payload.StudentIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "student_ids is required",
		})
	}

	result, err := h.svc.BatchEnroll(c.Context(), classID, tenantID, schoolID, academicTermID, payload.StudentIDs)
	if err != nil {
		return mapClassError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// ─── Unenroll Student ─────────────────────────────────────────────────────

// UnenrollStudent handles POST /api/v1/classes/:id/unenroll/:studentId.
func (h *Handler) UnenrollStudent(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	classID := c.Params("id")
	studentID := c.Params("studentId")

	if classID == "" || studentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "class id and student id are required",
		})
	}

	// Use the provided academic term from query or resolve the current one
	academicTermID := c.Query("academic_term_id")
	if academicTermID == "" {
		academicYearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicYearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is set for this school.",
			})
		}

		academicTermID, err = h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicTermID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	if err := h.svc.UnenrollStudent(c.Context(), classID, studentID, tenantID, schoolID, academicTermID); err != nil {
		return mapClassError(c, err)
	}

	return c.JSON(fiber.Map{
		"code":    "ok",
		"message": "Student successfully unenrolled.",
	})
}

// ─── Get Available Students ───────────────────────────────────────────────

// GetAvailableStudents handles GET /api/v1/classes/:id/available-students.
func (h *Handler) GetAvailableStudents(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	classID := c.Params("id")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "class id is required",
		})
	}

	// Use the provided academic year/term from query or resolve the current one
	academicYearID := c.Query("academic_year_id")
	academicTermID := c.Query("academic_term_id")

	if academicYearID == "" && academicTermID == "" {
		var err error
		academicYearID, err = h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicYearID == "" {
			return c.JSON(AvailableStudentsResponse{Items: []AvailableStudent{}, Total: 0, Page: 1, Limit: 50})
		}
	}

	if academicTermID == "" && academicYearID != "" {
		var err error
		academicTermID, err = h.academicYearsSvc.GetCurrentAcademicTermID(c.Context(), academicYearID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if academicTermID == "" {
			return c.JSON(AvailableStudentsResponse{Items: []AvailableStudent{}, Total: 0, Page: 1, Limit: 50})
		}
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

	filter := AvailableStudentsFilter{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		ClassID:        classID,
		AcademicYearID: academicYearID,
		AcademicTermID: academicTermID,
		Search:         c.Query("search"),
		Page:           page,
		Limit:          limit,
	}

	result, err := h.svc.GetAvailableStudents(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// mapClassError maps domain errors to the spec's error response shape.
func mapClassError(c *fiber.Ctx, err error) error {
	if isErrEnrollmentConflict(err) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"code":    "ENROLLMENT_CONFLICT",
			"message": "Enrollment failed. One or more selected students were updated elsewhere. Please refresh and try again.",
		})
	}
	if isErrStudentNotInClass(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "STUDENT_NOT_IN_CLASS",
			"message": "The student is not enrolled in this class.",
		})
	}
	// For validation errors from FieldError, return 422
	if fe, ok := err.(*middleware.FieldError); ok {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": fe.Error(),
			"errors":  fe.FieldErrors(),
		})
	}
	return middleware.HTTPError(c, err)
}

// isErrEnrollmentConflict checks if the error chain contains ErrEnrollmentConflict.
func isErrEnrollmentConflict(err error) bool {
	for err != nil {
		if err == ErrEnrollmentConflict {
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			return false
		}
	}
	return false
}

// isErrStudentNotInClass checks if the error chain contains ErrStudentNotInClass.
func isErrStudentNotInClass(err error) bool {
	for err != nil {
		if err == ErrStudentNotInClass {
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			return false
		}
	}
	return false
}
