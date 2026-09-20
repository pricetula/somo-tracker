package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type AttendanceHandler struct {
	svc    services.AttendanceService
	logger *zap.Logger
}

func NewAttendanceHandler(svc services.AttendanceService, logger *zap.Logger) *AttendanceHandler {
	return &AttendanceHandler{svc: svc, logger: logger.With(zap.String("handler", "attendance"))}
}

type createAttendanceRequest struct {
	ClassTimetableSlotID string `json:"class_timetable_slot_id"`
	AttendanceDate       string `json:"attendance_date"`
	Records              []struct {
		StudentID string `json:"student_id"`
		Status    string `json:"status"`
		Remarks   string `json:"remarks,omitempty"`
	} `json:"records"`
}

// @Summary Record attendance for a timetable slot
// @Tags Attendance
// @Accept json
// @Produce json
// @Param body body createAttendanceRequest true "Attendance payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/attendance [post]
func (h *AttendanceHandler) CreateAttendance(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid school id", "errors": fiber.Map{}})
	}

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "user not authenticated", "errors": fiber.Map{}})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid user id", "errors": fiber.Map{}})
	}

	var req createAttendanceRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid body", "errors": fiber.Map{}})
	}

	if req.ClassTimetableSlotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "validation_error", "message": "class_timetable_slot_id is required", "errors": fiber.Map{"class_timetable_slot_id": []string{"required"}}})
	}
	if req.AttendanceDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "validation_error", "message": "attendance_date is required", "errors": fiber.Map{"attendance_date": []string{"required"}}})
	}
	if len(req.Records) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "validation_error", "message": "at least one attendance record is required", "errors": fiber.Map{"records": []string{"at least one record required"}}})
	}

	records := make([]services.AttendanceStudentRecord, 0, len(req.Records))
	for i, r := range req.Records {
		if r.StudentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "validation_error", "message": fmt.Sprintf("student_id is required for record %d", i), "errors": fiber.Map{"records": []string{fmt.Sprintf("record %d: student_id required", i)}}})
		}
		if r.Status == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "validation_error", "message": fmt.Sprintf("status is required for record %d", i), "errors": fiber.Map{"records": []string{fmt.Sprintf("record %d: status required", i)}}})
		}
		records = append(records, services.AttendanceStudentRecord{
			StudentID: r.StudentID,
			Status:    r.Status,
			Remarks:   r.Remarks,
		})
	}

	err = h.svc.CreateAttendance(c.Context(), schoolID, userID, services.CreateAttendanceRequest{
		ClassTimetableSlotID: req.ClassTimetableSlotID,
		AttendanceDate:       req.AttendanceDate,
		Records:              records,
	})
	if err != nil {
		h.logger.Error("create attendance failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"code": "created", "message": "attendance recorded"})
}
