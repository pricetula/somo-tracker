package api

import (
	"strconv"

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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found",
			"errors":  fiber.Map{},
		})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school id",
			"errors":  fiber.Map{},
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := c.Query("search")
	classIDStr := c.Query("class_id")
	var classID *uuid.UUID
	if classIDStr != "" {
		cid, err := uuid.Parse(classIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": "invalid class_id",
				"errors":  fiber.Map{},
			})
		}
		classID = &cid
	}

	resp, err := h.svc.ListStudents(c.Context(), schoolID, page, limit, search, classID)
	if err != nil {
		h.logger.Error("list students failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list students",
			"errors":  fiber.Map{},
		})
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found",
			"errors":  fiber.Map{},
		})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school id",
			"errors":  fiber.Map{},
		})
	}

	var req deleteStudentsRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid request body",
			"errors":  fiber.Map{"body": []string{"malformed json"}},
		})
	}
	if len(req.StudentIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "student_ids must be provided",
			"errors":  fiber.Map{"student_ids": []string{"required"}},
		})
	}

	studentIDs := make([]uuid.UUID, 0, len(req.StudentIDs))
	for _, s := range req.StudentIDs {
		sid, err := uuid.Parse(s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": "invalid student id",
				"errors":  fiber.Map{"student_ids": []string{"invalid uuid: " + s}},
			})
		}
		studentIDs = append(studentIDs, sid)
	}

	if err := h.svc.DeleteStudents(c.Context(), schoolID, studentIDs); err != nil {
		h.logger.Error("delete students failed", zap.Error(err))
		if err.Error()[:len("bad_request:")] == "bad_request:" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": err.Error()[len("bad_request:"):],
				"errors":  fiber.Map{},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to delete students",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "students_deleted",
		"message": "students deleted successfully",
		"errors":  fiber.Map{},
	})
}
