package api

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type EnrollmentsHandler struct {
	svc    services.EnrollmentsService
	logger *zap.Logger
}

func NewEnrollmentsHandler(svc services.EnrollmentsService, logger *zap.Logger) *EnrollmentsHandler {
	return &EnrollmentsHandler{
		svc:    svc,
		logger: logger.With(zap.String("handler", "enrollments")),
	}
}

type createEnrollmentItem struct {
	StudentID      string          `json:"student_id"`
	EnrollmentDate string          `json:"enrollment_date"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// CreateEnrollments handles batch enrollment for a class
// @Summary Enroll multiple students in class
// @Tags Enrollments
// @Produce json
// @Param id path string true "Class ID"
// @Param body body []createEnrollmentItem true "Enrollment items"
// @Success 201 {object} map[string]interface{}
// @Router /classes/{id}/enrollments [post]
func (h *EnrollmentsHandler) CreateEnrollments(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	classIDStr := c.Params("id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid class id", nil))
	}
	var items []createEnrollmentItem
	if err := c.Bind().Body(&items); err != nil {
		return WriteError(c, ErrBadRequest("invalid request body", nil))
	}
	reqs := make([]services.CreateEnrollmentRequest, 0, len(items))
	for _, it := range items {
		studentID, _ := uuid.Parse(it.StudentID)
		reqs = append(reqs, services.CreateEnrollmentRequest{
			StudentID:      studentID,
			EnrollmentDate: it.EnrollmentDate,
			Metadata:       it.Metadata,
		})
	}
	created, err := h.svc.CreateEnrollments(c.Context(), schoolID, classID, reqs)
	if err != nil {
		h.logger.Error("create enrollments failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to create enrollments"))
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"items": created})
}

// ListUnassignedStudents lists students without active enrollment for the school's current academic year
// @Summary List unassigned students
// @Tags Enrollments
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} map[string]interface{}
// @Router /students/unassigned [get]
func (h *EnrollmentsHandler) ListUnassignedStudents(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, _ := uuid.Parse(schoolIDStr)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	items, total, err := h.svc.ListUnassignedStudentsCurrentYear(c.Context(), schoolID, page, limit)
	if err != nil {
		h.logger.Error("list unassigned students failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to list unassigned students"))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"items": items, "total": total})
}

// ListEnrollmentsByClass lists enrollments for a specific class
// @Summary List enrollments for class
// @Tags Enrollments
// @Produce json
// @Param id path string true "Class ID"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} map[string]interface{}
// @Router /classes/{id}/enrollments [get]
func (h *EnrollmentsHandler) ListEnrollmentsByClass(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, _ := uuid.Parse(schoolIDStr)
	classIDStr := c.Params("id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid class id", nil))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	params := services.ListEnrollmentsParams{
		SchoolID:    schoolID,
		Page:        page,
		Limit:       limit,
		ClassRoomID: &classID,
	}
	items, total, err := h.svc.ListEnrollments(c.Context(), params)
	if err != nil {
		h.logger.Error("list enrollments failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to list enrollments"))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"items": items, "total": total})
}

// UpdateEnrollment updates an enrollment
// @Summary Update enrollment
// @Tags Enrollments
// @Produce json
// @Param id path string true "Enrollment ID"
// @Param body body services.UpdateEnrollmentRequest true "Update"
// @Success 200 {object} map[string]interface{}
// @Router /enrollments/{id} [patch]
func (h *EnrollmentsHandler) UpdateEnrollment(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid enrollment id", nil))
	}
	var req services.UpdateEnrollmentRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid request body", nil))
	}
	updated, err := h.svc.UpdateEnrollment(c.Context(), id, req)
	if err != nil {
		h.logger.Error("update enrollment failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to update enrollment"))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"item": updated})
}

// DeleteEnrollment removes an enrollment
// @Summary Delete enrollment
// @Tags Enrollments
// @Param id path string true "Enrollment ID"
// @Success 204
// @Router /enrollments/{id} [delete]
func (h *EnrollmentsHandler) DeleteEnrollment(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid enrollment id", nil))
	}
	if err := h.svc.DeleteEnrollment(c.Context(), id); err != nil {
		h.logger.Error("delete enrollment failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to delete enrollment"))
	}
	return c.SendStatus(fiber.StatusNoContent)
}
