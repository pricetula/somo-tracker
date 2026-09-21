package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

// @Summary List attendance sessions for admin dashboard
// @Tags Attendance
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Param search query string false "Search class/teacher/subject"
// @Param status query string false "Filter: SUBMITTED, IN_PROGRESS, MISSED"
// @Param date_from query string false "YYYY-MM-DD"
// @Param date_to query string false "YYYY-MM-DD"
// @Param grades query string false "Comma-separated grade labels"
// @Param streams query string false "Comma-separated stream names"
// @Param subjects query string false "Comma-separated subject names"
// @Param teachers query string false "Comma-separated teacher names"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/attendance/sessions [get]
func (h *AttendanceHandler) ListAttendanceSessions(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid school id", "errors": fiber.Map{}})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := c.Query("search", "")
	statusFilter := c.Query("status", "")
	dateFromStr := c.Query("date_from", "")
	dateToStr := c.Query("date_to", "")
	gradesStr := c.Query("grades", "")
	streamsStr := c.Query("streams", "")
	subjectsStr := c.Query("subjects", "")
	teachersStr := c.Query("teachers", "")

	var dateFrom, dateTo *time.Time
	if dateFromStr != "" {
		if d, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			dateFrom = &d
		}
	}
	if dateToStr != "" {
		if d, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = &d
		}
	}

	parseCSV := func(s string) []string {
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	params := services.ListAttendanceSessionsParams{
		SchoolID:      schoolID,
		Page:          page,
		Limit:         limit,
		Search:        search,
		StatusFilter:  statusFilter,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		GradeFilter:   parseCSV(gradesStr),
		StreamFilter:  parseCSV(streamsStr),
		SubjectFilter: parseCSV(subjectsStr),
		TeacherFilter: parseCSV(teachersStr),
	}

	sessions, total, err := h.svc.ListAttendanceSessions(c.Context(), params)
	if err != nil {
		h.logger.Error("list attendance sessions failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
	}

	items := make([]fiber.Map, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, fiber.Map{
			"slot_id":           s.SlotID.String(),
			"class_id":          s.ClassID.String(),
			"class_name":        s.ClassName,
			"grade":             s.Grade,
			"stream":            s.Stream,
			"subject":           s.Subject,
			"teacher_name":      s.TeacherName,
			"teacher_id":        s.TeacherID.String(),
			"day_of_week":       s.DayOfWeek,
			"time_slot_name":    s.TimeSlotName,
			"start_time":        s.StartTime,
			"end_time":          s.EndTime,
			"attendance_date":   s.AttendanceDate,
			"status":            s.Status,
			"total_students":    s.TotalStudents,
			"recorded_students": s.RecordedStudents,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
