package timetable

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes timetable HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts timetable routes on the given router.
// Timetable routes are scoped under /api/v1/schools/:schoolId/timetable
// and /api/v1/schools/:schoolId/classes/:classId/teachers.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	tt := router.Group("/api/v1/schools/:schoolId/timetable")
	tt.Post("/slots/bulk", h.requireAuth, h.BulkCreateSlots)
	tt.Get("/slots", h.requireAuth, h.ListSlots)

	teachers := router.Group("/api/v1/schools/:schoolId/classes/:classId/teachers")
	teachers.Post("/", h.requireAuth, h.AssignTeacher)
	teachers.Delete("/:userId", h.requireAuth, h.RemoveTeacher)
}

// requireAuth extracts session info from context locals.
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

// BulkCreateSlots handles POST /api/v1/schools/:schoolId/timetable/slots/bulk.
func (h *Handler) BulkCreateSlots(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Params("schoolId")
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id is required",
		})
	}

	var input BulkCreateTimetableSlotsInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.BulkSaveSlots(c.Context(), tenantID, schoolID, input); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":    "created",
		"message": "timetable slots saved successfully",
	})
}

// ListSlots handles GET /api/v1/schools/:schoolId/timetable/slots.
// Query params: classId, termId, teacherId.
func (h *Handler) ListSlots(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Params("schoolId")
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id is required",
		})
	}

	classID := c.Query("class_id")
	teacherID := c.Query("teacher_id")
	termID := c.Query("term_id")

	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "term_id is required",
		})
	}

	slots, err := h.svc.GetSlots(c.Context(), tenantID, classID, teacherID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"data": slots,
	})
}

// AssignTeacher handles POST /api/v1/schools/:schoolId/classes/:classId/teachers.
func (h *Handler) AssignTeacher(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Params("schoolId")
	classID := c.Params("classId")
	if schoolID == "" || classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id and class_id are required",
		})
	}

	var payload struct {
		UserID         string      `json:"user_id"`
		LearningAreaID *string     `json:"learning_area_id"`
		TeacherRole    TeacherRole `json:"teacher_role"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if payload.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "user_id is required",
		})
	}
	if payload.TeacherRole == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "teacher_role is required",
		})
	}

	input := ClassTeacherInput{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		ClassID:        classID,
		UserID:         payload.UserID,
		LearningAreaID: payload.LearningAreaID,
		TeacherRole:    payload.TeacherRole,
	}

	if err := h.svc.AssignTeacher(c.Context(), input); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":    "created",
		"message": "teacher assigned successfully",
	})
}

// RemoveTeacher handles DELETE /api/v1/schools/:schoolId/classes/:classId/teachers/:userId.
func (h *Handler) RemoveTeacher(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Params("schoolId")
	classID := c.Params("classId")
	userID := c.Params("userId")
	if schoolID == "" || classID == "" || userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school_id, class_id, and user_id are required",
		})
	}

	if err := h.svc.RemoveTeacher(c.Context(), tenantID, classID, userID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "deleted",
		"message": "teacher removed successfully",
	})
}
