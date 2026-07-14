package attendance

import (
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
	att.Get("/roster/:timetable_slot_id", middleware.RequireAuth, h.GetRoster)
	att.Post("/bulk", middleware.RequireAuth, h.BulkMark)
	att.Get("/dashboard", middleware.RequireAuth, h.AdminDashboard)
	att.Get("/students/:student_id", middleware.RequireAuth, h.StudentHistory)
	att.Put("/records/:id", middleware.RequireAuth, h.UpdateRecord)
	att.Get("/children/:student_id/summary", middleware.RequireAuth, h.ChildSummary)

	// Background task trigger — recompute attendance_term_summaries
	att.Post("/summaries/compute", middleware.RequireAuth, h.ComputeSummaries)
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "timetable_slot_id is required",
		})
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
func (h *Handler) AdminDashboard(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	date := c.Query("date")
	result, err := h.svc.GetAdminDashboard(c.Context(), tenantID, schoolID, date)
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "student_id is required",
		})
	}

	filter := StudentHistoryFilter{
		TermID:    c.Query("term_id"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "student_id is required",
		})
	}

	termID := c.Query("term_id")
	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "term_id is required",
		})
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
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "term_id is required",
		})
	}

	if _, err := uuid.Parse(payload.TermID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid term_id",
		})
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
