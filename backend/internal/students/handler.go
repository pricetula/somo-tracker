package students

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes student HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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

	filter := ListFilter{
		TenantID: tenantID,
		SchoolID: schoolID,
		Page:     page,
		Limit:    limit,
		Search:   c.Query("search"),
		ClassID:  c.Query("class_id"),
		Gender:   c.Query("gender"),
	}

	result, err := h.svc.ListStudents(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ─── Create ───────────────────────────────────────────────────────────────

// Create handles POST /api/v1/students.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		schoolID = c.Locals("school_id").(string)
	}
	if schoolID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "active school not set", nil)
	}

	var body CreateStudentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.FullName == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "full_name is required",
			map[string][]string{"full_name": {"Full name is required"}})
	}

	student, err := h.svc.Create(c.Context(), tenantID, schoolID, body)
	if err != nil {
		if errors.Is(err, ErrDuplicateUPI) {
			return writeError(c, fiber.StatusConflict, "duplicate_upi",
				"A student with this UPI number already exists.",
				map[string][]string{"upi_number": {"This UPI number is already in use"}})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateStudentResponse{ID: student.ID})
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

	return c.JSON(ListEnrollmentsResponse{Data: enrollments})
}
