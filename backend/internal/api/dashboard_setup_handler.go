package api

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type DashboardSetupHandler struct {
	service *services.SetupSummaryService
	logger  *zap.Logger
}

func NewDashboardSetupHandler(svc *services.SetupSummaryService, logger *zap.Logger) *DashboardSetupHandler {
	return &DashboardSetupHandler{service: svc, logger: logger}
}

func (h *DashboardSetupHandler) GetSetupSummary(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return WriteError(c, ErrUnauthorized("active_school_id not found in session"))
	}

	data, err := h.service.GetSetupSummary(c.Context(), schoolID)
	if err != nil {
		h.logger.Error("setup summary failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to load setup summary"))
	}

	return c.JSON(fiber.Map{
		"code":    "setup_summary_retrieved",
		"message": "Setup summary retrieved",
		"data":    data,
		"errors":  fiber.Map{},
	})
}
