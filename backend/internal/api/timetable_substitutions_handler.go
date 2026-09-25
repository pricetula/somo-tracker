package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"somotracker/backend/internal/services"
)

type TimetableSubstitutionsHandler struct {
	svc    services.TimetableSubstitutionsService
	logger *zap.Logger
}

func NewTimetableSubstitutionsHandler(svc services.TimetableSubstitutionsService, logger *zap.Logger) *TimetableSubstitutionsHandler {
	return &TimetableSubstitutionsHandler{
		svc:    svc,
		logger: logger.With(zap.String("handler", "timetable_substitutions")),
	}
}

type createSubstitutionRequest struct {
	ClassTimetableSlotID          string  `json:"class_timetable_slot_id"`
	SubstitutionDate              string  `json:"substitution_date"`
	OriginalTeacherMembershipID   string  `json:"original_teacher_membership_id"`
	SubstituteTeacherMembershipID *string `json:"substitute_teacher_membership_id,omitempty"`
	Status                        string  `json:"status,omitempty"`
	Reason                        *string `json:"reason,omitempty"`
}

type updateSubstitutionRequest struct {
	ID                            string  `json:"id"`
	SubstituteTeacherMembershipID *string `json:"substitute_teacher_membership_id,omitempty"`
	Status                        *string `json:"status,omitempty"`
	Reason                        *string `json:"reason,omitempty"`
}

// @Summary Create timetable substitution
// @Tags Timetable
// @Accept json
// @Produce json
// @Param body body createSubstitutionRequest true "Substitution payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/timetable/substitutions [post]
func (h *TimetableSubstitutionsHandler) CreateSubstitution(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	var req createSubstitutionRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid body", nil))
	}
	if req.ClassTimetableSlotID == "" || req.SubstitutionDate == "" || req.OriginalTeacherMembershipID == "" {
		return WriteError(c, ErrBadRequest("missing required fields", map[string][]string{
			"class_timetable_slot_id":        {"required"},
			"substitution_date":              {"required"},
			"original_teacher_membership_id": {"required"},
		}))
	}
	classSlotID, err := uuid.Parse(req.ClassTimetableSlotID)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid class_timetable_slot_id", map[string][]string{"class_timetable_slot_id": {"must be uuid"}}))
	}
	originalTeacherID, err := uuid.Parse(req.OriginalTeacherMembershipID)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid original_teacher_membership_id", map[string][]string{"original_teacher_membership_id": {"must be uuid"}}))
	}
	var substituteTeacherID *uuid.UUID
	if req.SubstituteTeacherMembershipID != nil {
		id, err := uuid.Parse(*req.SubstituteTeacherMembershipID)
		if err != nil {
			return WriteError(c, ErrBadRequest("invalid substitute_teacher_membership_id", map[string][]string{"substitute_teacher_membership_id": {"must be uuid"}}))
		}
		substituteTeacherID = &id
	}
	id, err := h.svc.CreateSubstitution(c.Context(), schoolID, classSlotID, req.SubstitutionDate, originalTeacherID, substituteTeacherID, req.Status, req.Reason)
	if err != nil {
		h.logger.Error("create substitution failed", zap.Error(err))
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":    "substitution_created",
		"message": "Timetable substitution created",
		"id":      id,
		"errors":  fiber.Map{},
	})
}

// @Summary Update timetable substitution
// @Tags Timetable
// @Accept json
// @Produce json
// @Param body body updateSubstitutionRequest true "Update payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/timetable/substitutions [patch]
func (h *TimetableSubstitutionsHandler) UpdateSubstitution(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	var req updateSubstitutionRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid body", nil))
	}
	if req.ID == "" {
		return WriteError(c, ErrBadRequest("id is required", map[string][]string{"id": {"required"}}))
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid id", map[string][]string{"id": {"must be uuid"}}))
	}
	var substituteTeacherID *uuid.UUID
	if req.SubstituteTeacherMembershipID != nil {
		uid, err := uuid.Parse(*req.SubstituteTeacherMembershipID)
		if err != nil {
			return WriteError(c, ErrBadRequest("invalid substitute_teacher_membership_id", map[string][]string{"substitute_teacher_membership_id": {"must be uuid"}}))
		}
		substituteTeacherID = &uid
	}
	if err := h.svc.UpdateSubstitution(c.Context(), schoolID, id, substituteTeacherID, req.Status, req.Reason); err != nil {
		h.logger.Error("update substitution failed", zap.Error(err))
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "substitution_updated",
		"message": "Timetable substitution updated",
		"errors":  fiber.Map{},
	})
}

// @Summary Delete timetable substitution
// @Tags Timetable
// @Produce json
// @Param id path string true "Substitution ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/timetable/substitutions/{id} [delete]
func (h *TimetableSubstitutionsHandler) DeleteSubstitution(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	idStr := c.Params("id")
	if idStr == "" {
		return WriteError(c, ErrBadRequest("id is required", nil))
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid id", map[string][]string{"id": {"must be uuid"}}))
	}
	if err := h.svc.DeleteSubstitution(c.Context(), id); err != nil {
		h.logger.Error("delete substitution failed", zap.Error(err))
		return WriteError(c, ErrBadRequest(err.Error(), nil))
	}
	return c.SendStatus(fiber.StatusNoContent)
}
