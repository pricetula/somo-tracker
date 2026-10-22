package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"somotracker/backend/internal/api/middleware"
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
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	TimeSlots   []struct {
		Name            string `json:"name"`
		StartTime       string `json:"start_time"`
		EndTime         string `json:"end_time"`
		IsInstructional bool   `json:"is_instructional"`
	} `json:"time_slots"`
}

// @Summary List timetable templates
// @Tags Timetable
// @Produce json
// @Success 200 {array} object
// @Failure 401 {object} map[string]interface{}
// @Router /api/timetable/templates [get]
func (h *TimetableHandler) ListTemplates(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	pagination, err := ValidatePagination(c, 100)
	if err != nil {
		return err
	}
	items, total, err := h.svc.ListTemplates(c.Context(), schoolID, pagination.Page, pagination.Limit)
	if err != nil {
		h.logger.Error("list timetable templates failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to list timetable templates"))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"items": items,
		"total": total,
		"page":  pagination.Page,
		"limit": pagination.Limit,
	})
}

func (h *TimetableHandler) UpdateTemplate(c fiber.Ctx) error {
	templateIDStr := c.Params("id")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid template id", nil))
	}
	var req struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid body", nil))
	}
	if req.Name == "" {
		return WriteError(c, ErrBadRequest("name is required", map[string][]string{"name": {"name is required"}}))
	}
	if err := h.svc.UpdateTemplate(c.Context(), templateID, req.Name, req.Description); err != nil {
		h.logger.Error("update timetable template failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to update timetable template"))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "template updated"})
}

func (h *TimetableHandler) GetTemplate(c fiber.Ctx) error {
	templateIDStr := c.Params("id")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid template id", nil))
	}
	template, err := h.svc.GetTemplate(c.Context(), templateID)
	if err != nil {
		h.logger.Error("get timetable template failed", zap.Error(err))
		if errors.Is(err, pgx.ErrNoRows) {
			return WriteError(c, ErrNotFound("template not found"))
		}
		return WriteError(c, ErrInternal("failed to get template"))
	}
	return c.Status(fiber.StatusOK).JSON(template)
}

func (h *TimetableHandler) ListTimeSlotsByTemplate(c fiber.Ctx) error {
	templateIDStr := c.Params("id")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid template id", nil))
	}
	slots, err := h.svc.ListTimeSlotsByTemplate(c.Context(), templateID)
	if err != nil {
		h.logger.Error("list time slots failed", zap.Error(err))
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}
	result := make([]fiber.Map, 0, len(slots))
	for _, s := range slots {
		start := s.StartTime
		end := s.EndTime
		startStr := ""
		if start.Valid {
			startStr = fmt.Sprintf("%02d:%02d", start.Microseconds/3600000000, (start.Microseconds%3600000000)/60000000)
		}
		endStr := ""
		if end.Valid {
			endStr = fmt.Sprintf("%02d:%02d", end.Microseconds/3600000000, (end.Microseconds%3600000000)/60000000)
		}
		result = append(result, fiber.Map{
			"id":                    s.ID.String(),
			"timetable_template_id": s.TimetableTemplateID.String(),
			"name":                  s.Name,
			"start_time":            startStr,
			"end_time":              endStr,
			"sequence_index":        s.SequenceIndex,
			"is_instructional":      s.IsInstructional,
		})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *TimetableHandler) GetClassSlotsByTemplate(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	templateIDStr := c.Params("id")
	classIDStr := c.Params("classId")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid template id", nil))
	}
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid class id", nil))
	}
	// Verify class belongs to school (BOLA prevention)
	if err := h.svc.VerifyClassBelongsToSchool(c.Context(), schoolID, classID); err != nil {
		h.logger.Warn("class ownership verification failed", zap.Error(err), zap.String("school_id", schoolIDStr), zap.String("class_id", classIDStr))
		return WriteError(c, ErrNotFound("class not found"))
	}
	rows, err := h.svc.GetClassTimetableSlotsByTemplate(c.Context(), schoolID, classID, templateID)
	if err != nil {
		h.logger.Error("get class timetable slots failed", zap.Error(err))
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}
	return c.Status(fiber.StatusOK).JSON(rows)
}

// @Summary Delete a class timetable slot
// @Tags Timetable
// @Produce json
// @Param id path string true "Class timetable slot ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/timetable/class-timetable-slots/{id} [delete]
func (h *TimetableHandler) DeleteClassTimetableSlot(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	slotIDStr := c.Params("id")
	slotID, err := uuid.Parse(slotIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid slot id", nil))
	}
	if err := h.svc.DeleteClassTimetableSlot(c.Context(), schoolID, slotID); err != nil {
		h.logger.Error("delete class timetable slot failed", zap.Error(err))
		// Return 404 if slot not found or not in school (prevents enumeration)
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not in school") {
			return WriteError(c, ErrNotFound("slot not found"))
		}
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *TimetableHandler) SetupClassTimetableSlot(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	var req struct {
		ClassRoomID         string `json:"class_room_id"`
		DayOfWeek           int    `json:"day_of_week"`
		TimeSlotID          string `json:"time_slot_id"`
		SubjectID           string `json:"subject_id"`
		TeacherMembershipID string `json:"teacher_membership_id"`
		RoomID              string `json:"room_id"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid body", nil))
	}

	if req.ClassRoomID == "" || req.TimeSlotID == "" || req.SubjectID == "" || req.TeacherMembershipID == "" {
		return WriteError(c, ErrBadRequest("missing required fields", map[string][]string{
			"class_room_id":         {"required"},
			"time_slot_id":          {"required"},
			"subject_id":            {"required"},
			"teacher_membership_id": {"required"},
		}))
	}
	if req.DayOfWeek < 1 || req.DayOfWeek > 7 {
		return WriteError(c, ErrBadRequest("day_of_week must be 1-7", map[string][]string{"day_of_week": {"must be between 1 and 7"}}))
	}

	err = h.svc.SetupClassTimetableSlot(c.Context(), schoolID, req.ClassRoomID, req.TimeSlotID, req.SubjectID, req.TeacherMembershipID, req.DayOfWeek, req.RoomID)
	if err != nil {
		h.logger.Error("setup class timetable slot failed", zap.Error(err))
		if strings.HasPrefix(err.Error(), "timetable_conflict:") {
			conflictType := strings.TrimPrefix(err.Error(), "timetable_conflict:")
			// The feature spec requires structured conflicts array with objects.
			// APIError.Errors (map[string][]string) can't carry objects,
			// so return canonical JSON directly with structured conflicts.
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "timetable_conflict",
				"message": "Timetable conflict detected",
				"errors": map[string]interface{}{
					"conflicts": []map[string]interface{}{
						{
							"type":             conflictType,
							"resource_id":      "",
							"resource_name":    "",
							"existing_slot_id": "",
						},
					},
				},
				"request_id": middleware.GetRequestID(c),
			})
		}
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "class timetable slot created"})
}

// @Summary Create timetable template
// @Tags Timetable
// @Accept json
// @Produce json
// @Param body body createTimetableTemplateRequest true "Template payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/timetable/templates [post]
func (h *TimetableHandler) CreateTemplate(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}

	var req createTimetableTemplateRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid body", nil))
	}
	if req.Name == "" {
		return WriteError(c, ErrBadRequest("name is required", map[string][]string{"name": {"name is required"}}))
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
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}
