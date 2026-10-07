package api

import (
	"strconv"

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

	var req deleteFinanceRequest
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

	if err := h.svc.DeleteFinance(c.Context(), schoolID, userIDs, currentUserID); err != nil {
		h.logger.Error("delete finance failed", zap.Error(err))
		if err.Error()[:len("bad_request:")] == "bad_request:" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": err.Error()[len("bad_request:"):],
				"errors":  fiber.Map{},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to delete finance",
			"errors":  fiber.Map{},
		})
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

	resp, err := h.svc.ListFinance(c.Context(), schoolID, page, limit, search, invitationStatus)
	if err != nil {
		h.logger.Error("list finance failed", zap.Error(err))
		if err.Error()[:len("bad_request:")] == "bad_request:" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": err.Error()[len("bad_request:"):],
				"errors":  fiber.Map{},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list finance",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
