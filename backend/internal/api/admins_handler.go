package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type deleteAdminsRequest struct {
	UserIDs []string `json:"user_ids"`
}

type AdminsHandler struct {
	svc    services.AdminsService
	logger *zap.Logger
}

func NewAdminsHandler(svc services.AdminsService, logger *zap.Logger) *AdminsHandler {
	return &AdminsHandler{svc: svc, logger: logger.With(zap.String("handler", "admins"))}
}

// ListAdmins returns paginated admins for the active school with optional search and invitation status filter.
// @Summary List admins
// @Description List school admins with pagination and filters (invited/accepted)
// @Tags Admins
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Page size (default 50)"
// @Param search query string false "Search email or full name"
// @Param invitation_status query string false "Filter by invitation status: invited|accepted|all (default all)"
// @Success 200 {object} services.AdminListResponse
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /admins [get]
// DeleteAdmins removes members from the active school after validating self-deletion is prevented.
// @Summary Delete members
// @Description Delete school members by user IDs; returns error if current user is included
// @Tags Admins
// @Accept json
// @Produce json
// @Param body body deleteAdminsRequest true "User IDs to delete"
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /admins [delete]
func (h *AdminsHandler) DeleteAdmins(c fiber.Ctx) error {
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

	var req deleteAdminsRequest
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

	if err := h.svc.DeleteAdmins(c.Context(), schoolID, userIDs, currentUserID); err != nil {
		h.logger.Error("delete members failed", zap.Error(err))
		if err.Error()[:len("bad_request:")] == "bad_request:" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": err.Error()[len("bad_request:"):],
				"errors":  fiber.Map{},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to delete members",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "members_deleted",
		"message": "members deleted successfully",
		"errors":  fiber.Map{},
	})
}

func (h *AdminsHandler) ListAdmins(c fiber.Ctx) error {
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

	resp, err := h.svc.ListAdmins(c.Context(), schoolID, page, limit, search, invitationStatus)
	if err != nil {
		h.logger.Error("list admins failed", zap.Error(err))
		if err.Error()[:len("bad_request:")] == "bad_request:" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": err.Error()[len("bad_request:"):],
				"errors":  fiber.Map{},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list admins",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
