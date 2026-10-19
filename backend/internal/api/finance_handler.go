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
