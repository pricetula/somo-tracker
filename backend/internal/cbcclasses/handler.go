package cbcclasses

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes class HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts class routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	classes := router.Group("/api/v1/classes")
	classes.Get("/", h.requireAuth, h.List)
	classes.Post("/", h.requireAuth, h.Create)
	classes.Put("/:id", h.requireAuth, h.Update)
	classes.Delete("/", h.requireAuth, h.BulkDelete)
}

// ─── Auth middleware ───────────────────────────────────────────────────────

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	session := middleware.GetSession(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "authentication required",
		})
	}
	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	return c.Next()
}

// getActiveSchoolID extracts the active school_id from the request context.
// Checks multiple sources in order: query param, c.Locals("active_school_id"),
// and c.Locals("school_id") (set by the school-scoping middleware).
func getActiveSchoolID(c *fiber.Ctx) string {
	// Query param (explicit override)
	if schoolID := c.Query("school_id"); schoolID != "" {
		return schoolID
	}
	// Active school context (set by active-school middleware)
	if schoolID, ok := c.Locals("active_school_id").(string); ok && schoolID != "" {
		return schoolID
	}
	// School ID from scoping middleware (set by global middleware)
	if schoolID, ok := c.Locals("school_id").(string); ok && schoolID != "" {
		return schoolID
	}
	return ""
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/classes.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := getActiveSchoolID(c)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "academic_year_id is required",
		})
	}

	academicTermID := c.Query("academic_term_id")
	if academicTermID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "academic_term_id is required",
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

	filter := ClassListFilter{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: academicYearID,
		AcademicTermID: academicTermID,
		Page:           page,
		Limit:          limit,
	}

	if gradeLevel := c.Query("grade_level"); gradeLevel != "" {
		filter.GradeLevel = &gradeLevel
	}
	if streamID := c.Query("stream_id"); streamID != "" {
		filter.StreamID = &streamID
	}

	result, err := h.svc.ListClasses(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// Create handles POST /api/v1/classes.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := getActiveSchoolID(c)
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
	if payload.AcademicYearID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "academic_year_id is required",
			"errors":  map[string][]string{"academic_year_id": {"Academic year is required"}},
		})
	}
	if payload.AcademicTermID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "academic_term_id is required",
			"errors":  map[string][]string{"academic_term_id": {"Academic term is required"}},
		})
	}
	if payload.StreamID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "stream_id is required",
			"errors":  map[string][]string{"stream_id": {"Stream is required"}},
		})
	}

	if payload.StudentIDs == nil {
		payload.StudentIDs = []string{}
	}

	params := CreateClassParams{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: payload.AcademicYearID,
		AcademicTermID: payload.AcademicTermID,
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
	schoolID := getActiveSchoolID(c)
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
	if payload.AcademicTermID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "academic_term_id is required",
		})
	}

	if payload.StudentIDs == nil {
		payload.StudentIDs = []string{}
	}

	params := UpdateClassParams{
		ClassID:        classID,
		TenantID:       tenantID,
		SchoolID:       schoolID,
		GradeLevel:     payload.GradeLevel,
		StreamID:       payload.StreamID,
		AcademicTermID: payload.AcademicTermID,
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
	schoolID := getActiveSchoolID(c)
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

// mapClassError maps domain errors to the spec's error response shape.
func mapClassError(c *fiber.Ctx, err error) error {
	if isErrClassLocked(err) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "CLASS_LOCKED",
			"message": "This class has assessment records and cannot be modified.",
		})
	}
	if isErrClassHasAssessments(err) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "CLASS_HAS_ASSESSMENTS",
			"message": "One or more classes have assessment records and cannot be deleted.",
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

// isErrClassLocked checks if the error chain contains ErrClassLocked.
func isErrClassLocked(err error) bool {
	for err != nil {
		if err == ErrClassLocked {
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

// isErrClassHasAssessments checks if the error chain contains ErrClassHasAssessments.
func isErrClassHasAssessments(err error) bool {
	for err != nil {
		if err == ErrClassHasAssessments {
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
