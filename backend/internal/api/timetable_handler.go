package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type TimetableHandler struct {
	svc    services.TimetableService
	logger *zap.Logger
}

func NewTimetableHandler(svc services.TimetableService, logger *zap.Logger) *TimetableHandler {
	return &TimetableHandler{svc: svc, logger: logger.With(zap.String("handler", "timetable"))}
}

type createTimetableTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TimeSlots   []struct {
		Name            string `json:"name"`
		StartTime       string `json:"start_time"`
		EndTime         string `json:"end_time"`
		IsInstructional bool   `json:"is_instructional"`
	} `json:"time_slots"`
}

// @Summary Create timetable template
// @Tags Timetable
// @Produce json
// @Param body body createTimetableTemplateRequest true "Template payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /timetable/templates [post]
func (h *TimetableHandler) ListTemplates(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid school id", "errors": fiber.Map{}})
	}
	items, err := h.svc.ListTemplates(c.Context(), schoolID)
	if err != nil {
		h.logger.Error("list timetable templates failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusOK).JSON(items)
}

func (h *TimetableHandler) CreateTemplate(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid school id", "errors": fiber.Map{}})
	}

	var req createTimetableTemplateRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid body", "errors": fiber.Map{}})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "validation_error", "message": "name is required", "errors": fiber.Map{"name": []string{"name is required"}}})
	}

	slots := make([]services.CreateTimeSlot, 0, len(req.TimeSlots))
	for i, s := range req.TimeSlots {
		slots = append(slots, services.CreateTimeSlot{
			Name:            s.Name,
			StartTime:       s.StartTime,
			EndTime:         s.EndTime,
			IsInstructional: s.IsInstructional,
			SequenceIndex:   i,
		})
	}

	id, err := h.svc.CreateTemplate(c.Context(), schoolID, services.CreateTemplateRequest{
		Name:        req.Name,
		Description: req.Description,
		TimeSlots:   slots,
	})
	if err != nil {
		h.logger.Error("create timetable template failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}
