package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type deleteFinanceRequest struct {
	UserIDs []string `json:"user_ids"`
}

type FinanceHandler struct {
	svc    services.FinanceService
	logger *zap.Logger
}

func NewFinanceHandler(svc services.FinanceService, logger *zap.Logger) *FinanceHandler {
	return &FinanceHandler{svc: svc, logger: logger.With(zap.String("handler", "finance"))}
}

// DeleteFinance removes finance members from the active school; requires ADMIN role.
// @Summary Delete finance members
// @Description Delete school finance members by user IDs. Requires ADMIN role in the active school. Returns error if current user is included.
// @Tags Finance
// @Accept json
// @Produce json
// @Param body body deleteFinanceRequest true "User IDs to delete"
// @Success 200 {object} map[string]any "finance deleted successfully"
// @Failure 400 {object} map[string]any "bad request"
// @Failure 401 {object} map[string]any "unauthorized"
// @Failure 403 {object} map[string]any "forbidden - admin role required"
// @Failure 500 {object} map[string]any "internal error"
// @Security ApiKeyAuth
// @Router /finance [delete]
func (h *FinanceHandler) DeleteFinance(c fiber.Ctx) error {
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

	// Role check: only ADMIN can delete finance records
	hasAdmin, err := h.svc.UserHasAdminRole(c.Context(), schoolID, currentUserID)
	if err != nil {
		h.logger.Error("admin role check failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to verify permissions"))
	}
	if !hasAdmin {
		return WriteError(c, ErrForbidden("insufficient_permissions", "insufficient permissions"))
	}

	var req deleteFinanceRequest
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

	if err := h.svc.DeleteFinance(c.Context(), schoolID, userIDs, currentUserID); err != nil {
		h.logger.Error("delete finance failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to delete finance"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "finance_deleted",
		"message": "finance deleted successfully",
		"errors":  fiber.Map{},
	})
}

func (h *FinanceHandler) ListFinance(c fiber.Ctx) error {
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

	resp, err := h.svc.ListFinance(c.Context(), schoolID, page, limit, search, invitationStatus)
	if err != nil {
		h.logger.Error("list finance failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to list finance"))
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
