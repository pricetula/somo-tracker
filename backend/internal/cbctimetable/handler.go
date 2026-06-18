package cbctimetable

import (
	"context"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler exposes CBC timetable HTTP endpoints.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// RegisterRoutes mounts CBC timetable routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Mount under /api/v1/cbc
	cbc := router.Group("/api/v1/cbc")

	// Slots CRUD
	cbc.Get("/classes/:classId/timetable", h.FetchSlots)
	cbc.Post("/classes/:classId/timetable", h.CreateSlot)
	cbc.Put("/timetable/:slotId", h.UpdateSlot)
	cbc.Delete("/timetable/:slotId", h.DeleteSlot)

	// Conflict pre-check
	cbc.Get("/timetable/conflicts", h.CheckConflicts)

	// Bulk operations
	cbc.Post("/timetable/duplicate-day", h.DuplicateDay)
	cbc.Post("/timetable/copy-from-class", h.CopyFromClass)

	// Reference data
	cbc.Get("/learning-areas", h.FetchLearningAreas)
	cbc.Get("/teachers", h.FetchTeachers)
	cbc.Get("/class-teachers", h.FetchClassTeachers)
	cbc.Get("/room-autocomplete", h.FetchRoomAutocomplete)
	cbc.Get("/operating-days", h.FetchOperatingDays)

	// Slot metadata
	cbc.Get("/timetable/:slotId/attendance-count", h.FetchSlotAttendanceCount)

	// Attendance helpers
	cbc.Get("/classes/:classId/students", h.FetchClassStudents)
	cbc.Get("/attendance/slots/today", h.FetchTodayTeacherSlots)

	// ── Attendance periods ──────────────────────────────────────────
	cbc.Get("/classes/:classId/attendance/periods", h.FetchAttendancePeriods)
	cbc.Post("/classes/:classId/attendance/periods", h.CreateAttendancePeriod)
	cbc.Get("/attendance/periods/:periodId", h.FetchAttendancePeriodDetail)
	cbc.Get("/attendance/periods/:periodId/logs", h.FetchAttendanceLogs)

	// ── Attendance logs ─────────────────────────────────────────────
	cbc.Post("/attendance/logs", h.SaveAttendanceLog)
	cbc.Post("/attendance/logs/batch", h.BatchSaveAttendanceLogs)

	// ── Attendance analytics ────────────────────────────────────────
	cbc.Get("/classes/:classId/attendance/heatmap", h.FetchAttendanceHeatmap)
	cbc.Get("/classes/:classId/attendance/gaps", h.FetchAttendanceGaps)
}

// ─── Helper: extract tenant/school/user from context ──────────────────────

func (h *Handler) getContext(c *fiber.Ctx) (tenantID, userID string, ok bool) {
	tenantID, _ = c.Locals("tenant_id").(string)
	userID, _ = c.Locals("user_id").(string)
	if tenantID == "" || userID == "" {
		return "", "", false
	}
	return tenantID, userID, true
}

// ─── Slot CRUD handlers ───────────────────────────────────────────────────

// FetchSlots handles GET /api/v1/cbc/classes/:classId/timetable
func (h *Handler) FetchSlots(c *fiber.Ctx) error {
	classID := c.Params("classId")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "class_id is required"})
	}

	slots, err := h.svc.FetchSlots(c.Context(), classID)
	if err != nil {
		h.log.Error("fetch slots", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch slots"})
	}

	return c.JSON(slots)
}

// CreateSlot handles POST /api/v1/cbc/classes/:classId/timetable
func (h *Handler) CreateSlot(c *fiber.Ctx) error {
	classID := c.Params("classId")
	tenantID, _, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req CreateSlotRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	req.ClassID = classID

	// Resolve school_id from the class
	schoolID, err := h.resolveSchoolID(c.Context(), classID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	slot, err := h.svc.CreateSlot(c.Context(), schoolID, tenantID, &req)
	if err != nil {
		if err == ErrOverlap || strings.Contains(err.Error(), ErrOverlap.Error()) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "time_overlap",
				"message": "This slot conflicts with an existing timetable entry. Check the highlighted conflict.",
			})
		}
		if err == ErrInvalidDayOfWeek || err == ErrInvalidTimeRange {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		h.log.Error("create slot", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create slot"})
	}

	return c.Status(fiber.StatusCreated).JSON(slot)
}

// UpdateSlot handles PUT /api/v1/cbc/timetable/:slotId
func (h *Handler) UpdateSlot(c *fiber.Ctx) error {
	slotID := c.Params("slotId")

	var req UpdateSlotRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Resolve school ID from the existing slot
	existing, err := h.svc.repo.FetchSlotByID(c.Context(), slotID)
	if err != nil || existing == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "slot not found"})
	}

	updated, err := h.svc.UpdateSlot(c.Context(), slotID, existing.SchoolID, &req)
	if err != nil {
		if err == ErrOverlap || strings.Contains(err.Error(), ErrOverlap.Error()) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "time_overlap",
				"message": "This slot conflicts with an existing timetable entry.",
			})
		}
		if err == ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "slot not found"})
		}
		h.log.Error("update slot", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update slot"})
	}

	return c.JSON(updated)
}

// DeleteSlot handles DELETE /api/v1/cbc/timetable/:slotId
func (h *Handler) DeleteSlot(c *fiber.Ctx) error {
	slotID := c.Params("slotId")

	if err := h.svc.DeleteSlot(c.Context(), slotID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "slot not found"})
		}
		h.log.Error("delete slot", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete slot"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ─── Conflict pre-check handler ───────────────────────────────────────────

// CheckConflicts handles GET /api/v1/cbc/timetable/conflicts
func (h *Handler) CheckConflicts(c *fiber.Ctx) error {
	req := ConflictCheckRequest{
		TeacherID:      c.Query("teacher_id"),
		SchoolID:       c.Query("school_id"),
		AcademicYearID: c.Query("academic_year_id"),
		RoomIdentifier: stringPtr(c.Query("room_identifier")),
		ExcludeSlotID:  stringPtr(c.Query("exclude_slot_id")),
		ExcludeClassID: stringPtr(c.Query("exclude_class_id")),
	}

	dayOfWeek, err := strconv.Atoi(c.Query("day_of_week"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid day_of_week"})
	}
	req.DayOfWeek = dayOfWeek
	req.StartTime = c.Query("start_time")
	req.EndTime = c.Query("end_time")

	if req.TeacherID == "" || req.StartTime == "" || req.EndTime == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "teacher_id, start_time, and end_time are required"})
	}

	conflicts, err := h.svc.CheckConflicts(c.Context(), &req)
	if err != nil {
		h.log.Error("check conflicts", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to check conflicts"})
	}

	return c.JSON(conflicts)
}

// ─── Bulk operations ──────────────────────────────────────────────────────

// DuplicateDay handles POST /api/v1/cbc/timetable/duplicate-day
func (h *Handler) DuplicateDay(c *fiber.Ctx) error {
	var req DuplicateDayRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(req.TargetDays) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "at least one target day is required"})
	}

	result, err := h.svc.DuplicateDay(c.Context(), &req)
	if err != nil {
		h.log.Error("duplicate day", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to duplicate day"})
	}

	return c.JSON(result)
}

// CopyFromClass handles POST /api/v1/cbc/timetable/copy-from-class
func (h *Handler) CopyFromClass(c *fiber.Ctx) error {
	var req CopyFromClassRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.svc.CopyFromClass(c.Context(), &req)
	if err != nil {
		h.log.Error("copy from class", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to copy timetable"})
	}

	return c.JSON(result)
}

// ─── Reference data handlers ──────────────────────────────────────────────

// FetchLearningAreas handles GET /api/v1/cbc/learning-areas?grade_id=...
func (h *Handler) FetchLearningAreas(c *fiber.Ctx) error {
	gradeID := c.Query("grade_id")
	if gradeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "grade_id is required"})
	}

	areas, err := h.svc.FetchLearningAreas(c.Context(), gradeID)
	if err != nil {
		h.log.Error("fetch learning areas", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch learning areas"})
	}

	return c.JSON(areas)
}

// FetchTeachers handles GET /api/v1/cbc/teachers?school_id=...
func (h *Handler) FetchTeachers(c *fiber.Ctx) error {
	schoolID := c.Query("school_id")
	tenantID, _, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teachers, err := h.svc.FetchTeachers(c.Context(), schoolID, tenantID)
	if err != nil {
		h.log.Error("fetch teachers", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch teachers"})
	}

	return c.JSON(teachers)
}

// FetchClassTeachers handles GET /api/v1/cbc/class-teachers?class_id=...
func (h *Handler) FetchClassTeachers(c *fiber.Ctx) error {
	classID := c.Query("class_id")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "class_id is required"})
	}

	var learningAreaID *string
	if la := c.Query("learning_area_id"); la != "" {
		learningAreaID = &la
	}

	teachers, err := h.svc.FetchClassTeachers(c.Context(), classID, learningAreaID)
	if err != nil {
		h.log.Error("fetch class teachers", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch class teachers"})
	}

	return c.JSON(teachers)
}

// FetchRoomAutocomplete handles GET /api/v1/cbc/room-autocomplete?q=...
func (h *Handler) FetchRoomAutocomplete(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON([]string{})
	}

	tenantID, _, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	schoolID := c.Query("school_id")

	rooms, err := h.svc.FetchRoomAutocomplete(c.Context(), query, schoolID, tenantID)
	if err != nil {
		h.log.Error("fetch room autocomplete", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rooms"})
	}

	return c.JSON(rooms)
}

// FetchOperatingDays handles GET /api/v1/cbc/operating-days?school_id=...
func (h *Handler) FetchOperatingDays(c *fiber.Ctx) error {
	schoolID := c.Query("school_id")
	tenantID, _, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	days, err := h.svc.FetchOperatingDays(c.Context(), schoolID, tenantID)
	if err != nil {
		h.log.Error("fetch operating days", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch operating days"})
	}

	type dayObj struct {
		Value      int    `json:"value"`
		Label      string `json:"label"`
		ShortLabel string `json:"short_label"`
	}
	dayLabels := map[int]string{1: "Monday", 2: "Tuesday", 3: "Wednesday", 4: "Thursday", 5: "Friday", 6: "Saturday", 7: "Sunday"}
	dayShort := map[int]string{1: "Mon", 2: "Tue", 3: "Wed", 4: "Thu", 5: "Fri", 6: "Sat", 7: "Sun"}

	result := make([]dayObj, 0, len(days))
	for _, d := range days {
		label := dayLabels[d]
		if label == "" {
			label = "Unknown"
		}
		result = append(result, dayObj{Value: d, Label: label, ShortLabel: dayShort[d]})
	}

	return c.JSON(result)
}

// FetchSlotAttendanceCount handles GET /api/v1/cbc/timetable/:slotId/attendance-count
func (h *Handler) FetchSlotAttendanceCount(c *fiber.Ctx) error {
	slotID := c.Params("slotId")
	if slotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slot_id is required"})
	}

	count, err := h.svc.FetchSlotAttendanceCount(c.Context(), slotID)
	if err != nil {
		h.log.Error("fetch slot attendance count", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch attendance count"})
	}

	return c.JSON(count)
}

// ─── Attendance helpers ───────────────────────────────────────────────────

// FetchClassStudents handles GET /api/v1/cbc/classes/:classId/students?academic_term_id=...
func (h *Handler) FetchClassStudents(c *fiber.Ctx) error {
	classID := c.Params("classId")
	termID := c.Query("academic_term_id")
	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "academic_term_id is required"})
	}

	students, err := h.svc.FetchClassStudents(c.Context(), classID, termID)
	if err != nil {
		h.log.Error("fetch class students", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch students"})
	}

	return c.JSON(students)
}

// FetchTodayTeacherSlots handles GET /api/v1/cbc/attendance/slots/today?teacher_id=...
func (h *Handler) FetchTodayTeacherSlots(c *fiber.Ctx) error {
	teacherID := c.Query("teacher_id")
	if teacherID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "teacher_id is required"})
	}

	slots, err := h.svc.FetchTodayTeacherSlots(c.Context(), teacherID)
	if err != nil {
		h.log.Error("fetch today teacher slots", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch today's slots"})
	}

	return c.JSON(slots)
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — period handlers
// ═══════════════════════════════════════════════════════════════════════════

// FetchAttendancePeriods handles GET /api/v1/cbc/classes/:classId/attendance/periods
// Query params: ?date=YYYY-MM-DD or ?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) FetchAttendancePeriods(c *fiber.Ctx) error {
	classID := c.Params("classId")
	if classID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "class_id is required"})
	}

	date := c.Query("date")
	from := c.Query("from")
	to := c.Query("to")

	// Single-date fetch
	if date != "" {
		periods, err := h.svc.FetchAttendancePeriodsByDate(c.Context(), classID, date)
		if err != nil {
			h.log.Error("fetch attendance periods by date", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch periods"})
		}
		return c.JSON(periods)
	}

	// Date range fetch (summaries)
	if from != "" && to != "" {
		summaries, err := h.svc.FetchAttendancePeriodSummaries(c.Context(), classID, from, to)
		if err != nil {
			h.log.Error("fetch attendance period summaries", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch period summaries"})
		}
		return c.JSON(summaries)
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provide ?date= or ?from=&to="})
}

// CreateAttendancePeriod handles POST /api/v1/cbc/classes/:classId/attendance/periods
func (h *Handler) CreateAttendancePeriod(c *fiber.Ctx) error {
	classID := c.Params("classId")
	tenantID, userID, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req CreatePeriodRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.LearningAreaID == "" || req.DateRecorded == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cbc_learning_area_id and date_recorded are required"})
	}

	// Resolve school_id from the class
	schoolID, err := h.resolveSchoolID(c.Context(), classID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	period, err := h.svc.CreateAttendancePeriod(c.Context(), classID, tenantID, schoolID, userID, &req)
	if err != nil {
		if err == ErrNoCurrentTerm {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no current academic term set"})
		}
		// Unique constraint violation (duplicate period for same class/date/area)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "An attendance period already exists for this class, date, and learning area"})
		}
		h.log.Error("create attendance period", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create attendance period"})
	}

	return c.Status(fiber.StatusCreated).JSON(period)
}

// FetchAttendancePeriodDetail handles GET /api/v1/cbc/attendance/periods/:periodId
func (h *Handler) FetchAttendancePeriodDetail(c *fiber.Ctx) error {
	periodID := c.Params("periodId")
	if periodID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "period_id is required"})
	}

	summary, err := h.svc.FetchAttendancePeriodSummary(c.Context(), periodID)
	if err != nil {
		h.log.Error("fetch attendance period detail", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch period detail"})
	}
	if summary == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "period not found"})
	}

	return c.JSON(summary)
}

// FetchAttendanceLogs handles GET /api/v1/cbc/attendance/periods/:periodId/logs
func (h *Handler) FetchAttendanceLogs(c *fiber.Ctx) error {
	periodID := c.Params("periodId")
	if periodID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "period_id is required"})
	}

	logs, err := h.svc.FetchAttendanceLogs(c.Context(), periodID)
	if err != nil {
		h.log.Error("fetch attendance logs", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch attendance logs"})
	}

	return c.JSON(logs)
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — log handlers
// ═══════════════════════════════════════════════════════════════════════════

// SaveAttendanceLog handles POST /api/v1/cbc/attendance/logs
func (h *Handler) SaveAttendanceLog(c *fiber.Ctx) error {
	tenantID, userID, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req SaveLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.PeriodID == "" || req.StudentID == "" || req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cbc_attendance_period_id, student_id, and status are required"})
	}

	log, err := h.svc.SaveAttendanceLog(c.Context(), tenantID, userID, &req)
	if err != nil {
		h.log.Error("save attendance log", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save attendance"})
	}

	return c.JSON(log)
}

// BatchSaveAttendanceLogs handles POST /api/v1/cbc/attendance/logs/batch
func (h *Handler) BatchSaveAttendanceLogs(c *fiber.Ctx) error {
	tenantID, userID, ok := h.getContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req BatchSaveLogsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.PeriodID == "" || len(req.Marks) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cbc_attendance_period_id and marks are required"})
	}

	logs, err := h.svc.BatchSaveAttendanceLogs(c.Context(), tenantID, userID, &req)
	if err != nil {
		h.log.Error("batch save attendance logs", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save attendance"})
	}

	return c.JSON(logs)
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — analytics handlers
// ═══════════════════════════════════════════════════════════════════════════

// FetchAttendanceHeatmap handles GET /api/v1/cbc/classes/:classId/attendance/heatmap?term_id=...
func (h *Handler) FetchAttendanceHeatmap(c *fiber.Ctx) error {
	classID := c.Params("classId")
	termID := c.Query("term_id")
	if classID == "" || termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "class_id and term_id are required"})
	}

	days, err := h.svc.FetchAttendanceHeatmap(c.Context(), classID, termID)
	if err != nil {
		h.log.Error("fetch attendance heatmap", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch heatmap"})
	}

	return c.JSON(days)
}

// FetchAttendanceGaps handles GET /api/v1/cbc/classes/:classId/attendance/gaps?from=...&to=...
func (h *Handler) FetchAttendanceGaps(c *fiber.Ctx) error {
	classID := c.Params("classId")
	from := c.Query("from")
	to := c.Query("to")
	if classID == "" || from == "" || to == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "class_id, from, and to are required"})
	}

	gaps, err := h.svc.FetchAttendanceGaps(c.Context(), classID, from, to)
	if err != nil {
		h.log.Error("fetch attendance gaps", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch attendance gaps"})
	}

	return c.JSON(gaps)
}

// ─── Internal helpers ──────────────────────────────────────────────────────

func (h *Handler) resolveSchoolID(ctx context.Context, classID string) (string, error) {
	return h.svc.repo.ResolveSchoolID(ctx, classID)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
