package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"somotracker/backend/internal/services"
)

type AttendanceReportsHandler struct {
	svc    *services.AttendanceReportingService
	logger *zap.Logger
}

func NewAttendanceReportsHandler(svc *services.AttendanceReportingService, logger *zap.Logger) *AttendanceReportsHandler {
	return &AttendanceReportsHandler{svc: svc, logger: logger.With(zap.String("handler", "attendance_reports"))}
}

func (h *AttendanceReportsHandler) ListStudentTermStats(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		return WriteError(c, ErrBadRequest("term_id is required", map[string][]string{"term_id": {"required"}}))
	}
	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid term_id", nil))
	}
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		// parse int
	}
	items, _, err := h.svc.ListStudentTermStats(c.Context(), schoolID, termID, page, limit)
	if err != nil {
		h.logger.Error("list attendance stats failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to list attendance stats"))
	}
	return c.JSON(fiber.Map{"items": items})
}
