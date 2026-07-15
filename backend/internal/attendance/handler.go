package attendance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"somotracker/backend/internal/middleware"
)

// Handler exposes attendance HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new attendance Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts attendance routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	att := router.Group("/api/v1/attendance")

	// Teacher+Admin routes
	att.Get("/roster/:timetable_slot_id", middleware.RequireRole("TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.GetRoster)
	att.Post("/bulk", middleware.RequireRole("TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.BulkMark)
	att.Post("/sessions/skip", middleware.RequireRole("TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.SkipSession)
	att.Post("/sessions/unskip", middleware.RequireRole("TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.UnskipSession)
	att.Get("/sessions/:timetable_slot_id", middleware.RequireRole("TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.GetSession)

	// Admin-only routes
	att.Get("/dashboard", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.AdminDashboard)
	att.Get("/students/:student_id", middleware.RequireRole("TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.StudentHistory)
	att.Put("/records/:id", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.UpdateRecord)

	// Parent-facing route
	att.Get("/children/:student_id/summary", middleware.RequireRole("PARENT", "TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.ChildSummary)

	// Background task trigger — recompute attendance_term_summaries
	att.Post("/summaries/compute", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.ComputeSummaries)
}

// attMiddleware extracts common tenant/school from context.
func (h *Handler) attMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
	tenantID = c.Locals("tenant_id").(string)
	schoolID, _ = c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	return tenantID, schoolID, nil
}

// GetRoster handles GET /api/v1/attendance/roster/:timetable_slot_id.
func (h *Handler) GetRoster(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	timetableSlotID := c.Params("timetable_slot_id")
	if timetableSlotID == "" {
		return middleware.HTTPError(c, fmt.Errorf("timetable_slot_id is required: %w", middleware.ErrInvalidInput))
	}

	date := c.Query("date")
	if date == "" {
		// Default to today
		date = c.Context().Time().Format("2006-01-02")
	}

	result, err := h.svc.GetRosterForSlot(c.Context(), tenantID, schoolID, timetableSlotID, date)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// BulkMark handles POST /api/v1/attendance/bulk.
func (h *Handler) BulkMark(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "user not authenticated",
		})
	}

	var payload BulkAttendancePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if err := h.svc.BulkMarkAttendance(c.Context(), tenantID, schoolID, payload, userID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Attendance saved",
		"count":   len(payload.Entries),
	})
}

// AdminDashboard handles GET /api/v1/attendance/dashboard.
// Supports pagination and multi-value filters via repeated query params:
//
//	?education_level=Early_Years&education_level=Upper_Primary
//	&grade_level=G4&grade_level=G7
//	&class_id=<uuid>
//	&is_complete=complete|incomplete
//	&page=1&limit=50
func (h *Handler) AdminDashboard(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	date := c.Query("date")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	filter := DashboardFilter{
		EducationLevels: parseRepeatedQuery(c, "education_level"),
		GradeLevels:     parseRepeatedQuery(c, "grade_level"),
		ClassID:         c.Query("class_id"),
		IsComplete:      c.Query("is_complete"),
		Page:            page,
		Limit:           limit,
	}

	result, err := h.svc.GetAdminDashboard(c.Context(), tenantID, schoolID, date, filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// StudentHistory handles GET /api/v1/attendance/students/:student_id.
func (h *Handler) StudentHistory(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	studentID := c.Params("student_id")
	if studentID == "" {
		return middleware.HTTPError(c, fmt.Errorf("student_id is required: %w", middleware.ErrInvalidInput))
	}

	filter := StudentHistoryFilter{
		TermID:         c.Query("term_id"),
		StartDate:      c.Query("start_date"),
		EndDate:        c.Query("end_date"),
		LearningAreaID: c.Query("learning_area_id"),
	}

	records, err := h.svc.GetStudentHistory(c.Context(), tenantID, schoolID, studentID, filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"items": records,
		"total": len(records),
	})
}

// UpdateRecord handles PUT /api/v1/attendance/records/:id.
func (h *Handler) UpdateRecord(c *fiber.Ctx) error {
	tenantID, _, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	recordID := c.Params("id")
	if recordID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "record id is required",
		})
	}

	var payload UpdateAttendanceEntryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UpdateAttendanceRecord(c.Context(), recordID, tenantID, payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Attendance record updated"})
}

// ChildSummary handles GET /api/v1/attendance/children/:student_id/summary.
func (h *Handler) ChildSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	studentID := c.Params("student_id")
	if studentID == "" {
		return middleware.HTTPError(c, fmt.Errorf("student_id is required: %w", middleware.ErrInvalidInput))
	}

	termID := c.Query("term_id")
	if termID == "" {
		return middleware.HTTPError(c, fmt.Errorf("term_id is required: %w", middleware.ErrInvalidInput))
	}

	summary, err := h.svc.GetChildAttendanceSummary(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(summary)
}

// ComputeSummaries handles POST /api/v1/attendance/summaries/compute.
func (h *Handler) ComputeSummaries(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	var payload struct {
		TermID string `json:"term_id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, fmt.Errorf("invalid request body: %w", middleware.ErrInvalidInput))
	}

	if _, err := uuid.Parse(payload.TermID); err != nil {
		return middleware.HTTPError(c, fmt.Errorf("invalid term_id: %w", middleware.ErrInvalidInput))
	}

	count, err := h.svc.ComputeTermSummaries(c.Context(), tenantID, schoolID, payload.TermID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Attendance summaries computed",
		"count":   count,
	})
}

// SkipSession handles POST /api/v1/attendance/sessions/skip.
func (h *Handler) SkipSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	var payload SkipSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if err := h.svc.SkipSession(c.Context(), tenantID, schoolID, payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Session marked as skipped. Attendance records deleted.",
	})
}

// UnskipSession handles POST /api/v1/attendance/sessions/unskip.
func (h *Handler) UnskipSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	var payload struct {
		TimetableSlotID string `json:"timetable_slot_id"`
		Date            string `json:"date"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UnskipSession(c.Context(), tenantID, schoolID, payload.TimetableSlotID, payload.Date); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Session re-opened. You can now mark attendance again.",
	})
}

// GetSession handles GET /api/v1/attendance/sessions/:timetable_slot_id.
func (h *Handler) GetSession(c *fiber.Ctx) error {
	tenantID, _, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	timetableSlotID := c.Params("timetable_slot_id")
	if timetableSlotID == "" {
		return middleware.HTTPError(c, fmt.Errorf("timetable_slot_id is required: %w", middleware.ErrInvalidInput))
	}

	date := c.Query("date")
	if date == "" {
		return middleware.HTTPError(c, fmt.Errorf("date query parameter is required: %w", middleware.ErrInvalidInput))
	}

	session, err := h.svc.GetSessionBySlotDate(c.Context(), tenantID, timetableSlotID, date)
	if err != nil {
		// If not found, return a null session rather than a 404
		if errors.Is(err, ErrNotFound) {
			return c.JSON(fiber.Map{
				"session": nil,
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"session": session,
	})
}

// parseRepeatedQuery reads all values for a given query parameter name.
// Supports two styles:
//   - repeated params: ?education_level=Early_Years&education_level=Upper_Primary
//   - comma-separated: ?education_level=Early_Years,Upper_Primary
func parseRepeatedQuery(c *fiber.Ctx, name string) []string {
	// Check for repeated query params first
	all := c.Request().URI().QueryArgs().PeekMulti(name)
	if len(all) > 1 {
		result := make([]string, 0, len(all))
		for _, v := range all {
			s := strings.TrimSpace(string(v))
			if s != "" {
				result = append(result, s)
			}
		}
		return result
	}

	// Single value (or none) — could be comma-separated
	vals := c.Query(name, "")
	if vals == "" {
		return nil
	}

	parts := strings.Split(vals, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
