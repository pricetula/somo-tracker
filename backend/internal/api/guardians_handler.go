package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type deleteGuardiansRequest struct {
	UserIDs []string `json:"user_ids"`
}

type GuardiansHandler struct {
	svc    services.GuardiansService
	logger *zap.Logger
}

func NewGuardiansHandler(svc services.GuardiansService, logger *zap.Logger) *GuardiansHandler {
	return &GuardiansHandler{svc: svc, logger: logger.With(zap.String("handler", "guardians"))}
}

func (h *GuardiansHandler) DeleteGuardians(c fiber.Ctx) error {
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

	var req deleteGuardiansRequest
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

	if err := h.svc.DeleteGuardians(c.Context(), schoolID, userIDs, currentUserID); err != nil {
		h.logger.Error("delete guardians failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to delete guardians"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "guardians_deleted",
		"message": "guardians deleted successfully",
		"errors":  fiber.Map{},
	})
}

// GetGuardianSummary returns guardian count details for the active school.
// @Summary Guardian summary
// @Description Get guardian count summary total and without student
// @Tags Guardians
// @Produce json
// @Success 200 {object} services.GuardianSummary
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /guardians/summary [get]
func (h *GuardiansHandler) GetGuardianSummary(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found in session"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	summary, err := h.svc.GetGuardianSummary(c.Context(), schoolID)
	if err != nil {
		h.logger.Error("get guardian summary failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to get guardian summary"))
	}

	return c.Status(fiber.StatusOK).JSON(summary)
}

func (h *GuardiansHandler) ListGuardians(c fiber.Ctx) error {
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

	resp, err := h.svc.ListGuardians(c.Context(), schoolID, page, limit, search, invitationStatus)
	if err != nil {
		h.logger.Error("list guardians failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "bad_request:") {
			return WriteError(c, ErrBadRequest(err.Error()[len("bad_request:"):], nil))
		}
		return WriteError(c, ErrInternal("failed to list guardians"))
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
