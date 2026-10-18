package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type deleteTeachersRequest struct {
	UserIDs []string `json:"user_ids"`
}

type TeachersHandler struct {
	svc    services.TeachersService
	logger *zap.Logger
}

func NewTeachersHandler(svc services.TeachersService, logger *zap.Logger) *TeachersHandler {
	return &TeachersHandler{svc: svc, logger: logger.With(zap.String("handler", "teachers"))}
}

func (h *TeachersHandler) DeleteTeachers(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found in session",
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

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "user not found in session",
			"errors":  fiber.Map{},
		})
	}
	currentUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid user id",
			"errors":  fiber.Map{},
		})
	}

	var req deleteTeachersRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid request body",
			"errors":  fiber.Map{"body": []string{"malformed json"}},
		})
	}
	if len(req.UserIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "user_ids must be provided",
			"errors":  fiber.Map{"user_ids": []string{"required"}},
		})
	}

	userIDs := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, s := range req.UserIDs {
		uid, err := uuid.Parse(s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": "invalid user id",
				"errors":  fiber.Map{"user_ids": []string{"invalid uuid: " + s}},
			})
		}
		userIDs = append(userIDs, uid)
	}

	if err := h.svc.DeleteTeachers(c.Context(), schoolID, userIDs, currentUserID); err != nil {
		h.logger.Error("delete teachers failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to delete teachers"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "teachers_deleted",
		"message": "teachers deleted successfully",
		"errors":  fiber.Map{},
	})
}

// GetTeacherSummary returns teacher count details for the active school.
// @Summary Teacher summary
// @Description Get teacher count summary total and without timetable assignment
// @Tags Teachers
// @Produce json
// @Success 200 {object} services.TeacherSummary
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /teachers/summary [get]
func (h *TeachersHandler) GetTeacherSummary(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found in session",
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

	summary, err := h.svc.GetTeacherSummary(c.Context(), schoolID)
	if err != nil {
		h.logger.Error("get teacher summary failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to get teacher summary"))
	}

	return c.Status(fiber.StatusOK).JSON(summary)
}

func (h *TeachersHandler) ListTeachers(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found in session",
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
	invitationStatus := c.Query("invitation_status")

	resp, err := h.svc.ListTeachers(c.Context(), schoolID, page, limit, search, invitationStatus)
	if err != nil {
		h.logger.Error("list teachers failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to list teachers"))
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
