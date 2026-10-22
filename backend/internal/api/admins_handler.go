package api

import (
	"strings"

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
// @Param limit query int false "Page size (default 50, max 100)"
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
		return WriteError(c, ErrUnauthorized("active school not found in session"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return WriteError(c, ErrUnauthorized("user not found in session"))
	}
	currentUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid user id", nil))
	}

	var req deleteAdminsRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid request body", map[string][]string{"body": {"malformed json"}}))
	}
	if len(req.UserIDs) == 0 {
		return WriteError(c, ErrBadRequest("user_ids must be provided", map[string][]string{"user_ids": {"required"}}))
	}

	userIDs := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, s := range req.UserIDs {
		uid, err := uuid.Parse(s)
		if err != nil {
			return WriteError(c, ErrBadRequest("invalid user id", map[string][]string{"user_ids": {"invalid uuid: " + s}}))
		}
		userIDs = append(userIDs, uid)
	}

	if err := h.svc.DeleteAdmins(c.Context(), schoolID, userIDs, currentUserID); err != nil {
		h.logger.Error("delete members failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to delete members"))
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
		return WriteError(c, ErrUnauthorized("active school not found in session"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	pagination, err := ValidatePagination(c, 100)
	if err != nil {
		return err
	}
	page, limit := pagination.Page, pagination.Limit

	search := c.Query("search")
	invitationStatus := c.Query("invitation_status")

	resp, err := h.svc.ListAdmins(c.Context(), schoolID, page, limit, search, invitationStatus)
	if err != nil {
		h.logger.Error("list admins failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to list admins"))
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
