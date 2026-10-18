package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type deleteStudentsRequest struct {
	StudentIDs []string `json:"student_ids"`
}

type StudentsHandler struct {
	svc    services.StudentsService
	logger *zap.Logger
}

func NewStudentsHandler(svc services.StudentsService, logger *zap.Logger) *StudentsHandler {
	return &StudentsHandler{svc: svc, logger: logger.With(zap.String("handler", "students"))}
}

// ListStudents returns paginated students for the active school with optional search.
// @Summary List students
// @Description List students with pagination and search
// @Tags Students
// @Produce json
// @Param page query int false "Page number default 1"
// @Param limit query int false "Page size default 50"
// @Param search query string false "Search full name or admission number"
// @Success 200 {object} services.StudentListResponse
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /students [get]
func (h *StudentsHandler) ListStudents(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "50")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return WriteError(c, ErrBadRequest("invalid page parameter", map[string][]string{"page": {"must be a positive integer"}}))
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		return WriteError(c, ErrBadRequest("invalid limit parameter", map[string][]string{"limit": {"must be between 1 and 200"}}))
	}
	search := c.Query("search")
	classIDStr := c.Query("class_id")
	var classID *uuid.UUID
	if classIDStr != "" {
		cid, err := uuid.Parse(classIDStr)
		if err != nil {
			return WriteError(c, ErrBadRequest("invalid class_id", map[string][]string{"class_id": {"invalid uuid"}}))
		}
		classID = &cid
	}
	withoutClass := c.Query("without_class") == "true"
	withoutGuardian := c.Query("without_guardian") == "true"

	resp, err := h.svc.ListStudents(c.Context(), schoolID, page, limit, search, classID, withoutClass, withoutGuardian)
	if err != nil {
		h.logger.Error("list students failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to list students"))
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// DeleteStudents removes students by their IDs from the active school.
// @Summary Delete students
// @Description Delete students by student IDs
// @Tags Students
// @Accept json
// @Produce json
// @Param body body deleteStudentsRequest true "Student IDs to delete"
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /students [delete]
func (h *StudentsHandler) DeleteStudents(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	var req deleteStudentsRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid request body", map[string][]string{"body": {"malformed json"}}))
	}
	if len(req.StudentIDs) == 0 {
		return WriteError(c, ErrBadRequest("student_ids must be provided", map[string][]string{"student_ids": {"required"}}))
	}

	studentIDs := make([]uuid.UUID, 0, len(req.StudentIDs))
	for i, s := range req.StudentIDs {
		sid, err := uuid.Parse(s)
		if err != nil {
			return WriteError(c, ErrBadRequest("invalid student id", map[string][]string{"student_ids": {fmt.Sprintf("invalid uuid at index %d: %s", i, s)}}))
		}
		studentIDs = append(studentIDs, sid)
	}

	if err := h.svc.DeleteStudents(c.Context(), schoolID, studentIDs); err != nil {
		h.logger.Error("delete students failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to delete students"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "students_deleted",
		"message": "students deleted successfully",
		"errors":  fiber.Map{},
	})
}

// GetStudentSummary returns student count details for the active school and term.
// @Summary Student summary
// @Description Get student count summary by gender and enrollment/guardian status
// @Tags Students
// @Produce json
// @Success 200 {object} services.StudentSummary
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /students/summary [get]
func (h *StudentsHandler) GetStudentSummary(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	summary, err := h.svc.GetStudentSummary(c.Context(), schoolID)
	if err != nil {
		h.logger.Error("get student summary failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to get student summary"))
	}

	return c.Status(fiber.StatusOK).JSON(summary)
}

// GetStudent returns enriched profile for a student
// @Summary Get student profile
// @Tags Students
// @Produce json
// @Param id path string true "Student ID"
// @Success 200 {object} services.StudentProfile
// @Security ApiKeyAuth
// @Router /students/{id} [get]
func (h *StudentsHandler) GetStudent(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid student id", nil))
	}
	profile, err := h.svc.GetStudentByID(c.Context(), schoolID, studentID)
	if err != nil {
		h.logger.Error("get student failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to get student"))
	}
	return c.Status(fiber.StatusOK).JSON(profile)
}

// UpdateStudent patches personal fields
// @Summary Update student profile
// @Tags Students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Param body body services.UpdateStudentRequest true "Update payload"
// @Success 200 {object} map[string]any
// @Security ApiKeyAuth
// @Router /students/{id} [patch]
func (h *StudentsHandler) UpdateStudent(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid student id", nil))
	}
	var req services.UpdateStudentRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid body", nil))
	}
	if err := h.svc.UpdateStudent(c.Context(), schoolID, studentID, req); err != nil {
		h.logger.Error("update student failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to update student"))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"code": "student_updated", "message": "student updated", "errors": fiber.Map{}})
}
