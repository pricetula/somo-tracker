package attendance

import (
	"time"

	"github.com/gofiber/fiber/v2"

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
	// Attendance sessions
	sessions := router.Group("/api/v1/attendance/sessions")
	sessions.Post("/", middleware.RequireAuth, h.CreateSession)
	sessions.Get("/", middleware.RequireAuth, h.ListSessions)
	sessions.Get("/class/:class_id/date/:date", middleware.RequireAuth, h.GetSessionsForClassDate)
	sessions.Get("/:id", middleware.RequireAuth, h.GetSession)
	sessions.Put("/:id", middleware.RequireAuth, h.UpdateSession)

	// Attendance records
	records := router.Group("/api/v1/attendance/records")
	records.Post("/batch", middleware.RequireAuth, h.BatchMark)
	records.Get("/slot", middleware.RequireAuth, h.ListRecordsBySlotDate)
	records.Get("/student/:student_id", middleware.RequireAuth, h.ListRecordsByStudentTerm)
	records.Get("/class/:class_id/date/:date", middleware.RequireAuth, h.ListRecordsByClassDate)
	records.Get("/", middleware.RequireAuth, h.ListRecords)
	records.Put("/:id", middleware.RequireAuth, h.UpdateRecord)

	// Attendance summaries
	summaries := router.Group("/api/v1/attendance/summaries")
	summaries.Get("/student/:student_id", middleware.RequireAuth, h.GetStudentTermSummary)
	summaries.Get("/class/:class_id", middleware.RequireAuth, h.GetClassTermSummary)
	summaries.Post("/refresh", middleware.RequireAuth, h.RefreshSummaries)

	// Class daily attendance summaries
	daily := router.Group("/api/v1/attendance/daily")
	daily.Get("/class/:class_id/date/:date", middleware.RequireAuth, h.GetClassDailySummary)
	daily.Post("/class/:class_id/date/:date/refresh", middleware.RequireAuth, h.RefreshClassDailySummary)
	daily.Get("/class/:class_id", middleware.RequireAuth, h.ListClassDailySummaries)

	// Calendar status (monthly overview)
	calendar := router.Group("/api/v1/attendance/calendar")
	calendar.Get("/status", middleware.RequireAuth, h.GetCalendarStatus)
}

// attMiddleware extracts common tenant/school context.
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

// ── Sessions ──────────────────────────────────────────────────────────────

// CreateSession handles POST /api/v1/attendance/sessions.
func (h *Handler) CreateSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	var payload CreateSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	session, err := h.svc.CreateSession(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(session)
}

// ListSessions handles GET /api/v1/attendance/sessions.
func (h *Handler) ListSessions(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	filter := SessionFilter{
		TimetableSlotID: c.Query("timetable_slot_id"),
		Date:            c.Query("date"),
		Status:          c.Query("status"),
		ClassID:         c.Query("class_id"),
		SchoolID:        schoolID,
		TenantID:        tenantID,
	}

	result, err := h.svc.ListSessions(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetSession handles GET /api/v1/attendance/sessions/:id.
func (h *Handler) GetSession(c *fiber.Ctx) error {
	tenantID, _, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	session, err := h.svc.GetEnrichedSession(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(session)
}

// GetSessionsForClassDate handles GET /api/v1/attendance/sessions/class/:class_id/date/:date.
func (h *Handler) GetSessionsForClassDate(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	date := c.Params("date")

	result, err := h.svc.GetSessionsForClassDate(c.Context(), tenantID, schoolID, classID, date)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// UpdateSession handles PUT /api/v1/attendance/sessions/:id.
func (h *Handler) UpdateSession(c *fiber.Ctx) error {
	tenantID, _, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	var payload UpdateSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	session, err := h.svc.UpdateSession(c.Context(), id, tenantID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(session)
}

// ── Records ───────────────────────────────────────────────────────────────

// BatchMark handles POST /api/v1/attendance/records/batch.
func (h *Handler) BatchMark(c *fiber.Ctx) error {
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

	var payload BatchMarkPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Accept optional term_id in query or from request context
	termID := c.Query("term_id")

	result, err := h.svc.BatchMark(c.Context(), tenantID, schoolID, payload, userID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// ListRecordsBySlotDate handles GET /api/v1/attendance/records/slot.
func (h *Handler) ListRecordsBySlotDate(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	timetableSlotID := c.Query("timetable_slot_id")
	date := c.Query("date")

	result, err := h.svc.ListRecordsBySlotDate(c.Context(), tenantID, schoolID, timetableSlotID, date)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListRecordsByStudentTerm handles GET /api/v1/attendance/records/student/:student_id.
func (h *Handler) ListRecordsByStudentTerm(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	studentID := c.Params("student_id")
	termID := c.Query("term_id")

	result, err := h.svc.ListRecordsByStudentTerm(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListRecordsByClassDate handles GET /api/v1/attendance/records/class/:class_id/date/:date.
func (h *Handler) ListRecordsByClassDate(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	date := c.Params("date")
	termID := c.Query("term_id")

	result, err := h.svc.ListRecordsByClassDate(c.Context(), tenantID, schoolID, classID, date, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListRecords handles GET /api/v1/attendance/records.
func (h *Handler) ListRecords(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	filter := RecordFilter{
		TimetableSlotID: c.Query("timetable_slot_id"),
		Date:            c.Query("date"),
		StudentID:       c.Query("student_id"),
		ClassID:         c.Query("class_id"),
		AcademicTermID:  c.Query("academic_term_id"),
		Status:          c.Query("status"),
		SchoolID:        schoolID,
		TenantID:        tenantID,
	}

	result, err := h.svc.ListRecords(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// UpdateRecord handles PUT /api/v1/attendance/records/:id.
func (h *Handler) UpdateRecord(c *fiber.Ctx) error {
	tenantID, _, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	var payload UpdateRecordPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	record, err := h.svc.UpdateRecord(c.Context(), id, tenantID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(record)
}

// ── Summaries ─────────────────────────────────────────────────────────────

// GetStudentTermSummary handles GET /api/v1/attendance/summaries/student/:student_id.
func (h *Handler) GetStudentTermSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	studentID := c.Params("student_id")
	termID := c.Query("term_id")

	result, err := h.svc.GetStudentTermSummary(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetClassTermSummary handles GET /api/v1/attendance/summaries/class/:class_id.
func (h *Handler) GetClassTermSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	termID := c.Query("term_id")

	result, err := h.svc.GetClassTermSummary(c.Context(), tenantID, schoolID, classID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// RefreshSummaries handles POST /api/v1/attendance/summaries/refresh.
func (h *Handler) RefreshSummaries(c *fiber.Ctx) error {
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
			"message": "invalid request body",
		})
	}

	result, err := h.svc.RefreshSummaries(c.Context(), tenantID, schoolID, payload.TermID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ── Class Daily Summaries ─────────────────────────────────────────────────

// GetClassDailySummary handles GET /api/v1/attendance/daily/class/:class_id/date/:date.
func (h *Handler) GetClassDailySummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	date := c.Params("date")

	result, err := h.svc.GetClassDailySummary(c.Context(), tenantID, schoolID, classID, date)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// RefreshClassDailySummary handles POST /api/v1/attendance/daily/class/:class_id/date/:date/refresh.
func (h *Handler) RefreshClassDailySummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	date := c.Params("date")

	result, err := h.svc.RefreshClassDailySummary(c.Context(), tenantID, schoolID, classID, date)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListClassDailySummaries handles GET /api/v1/attendance/daily/class/:class_id.
func (h *Handler) ListClassDailySummaries(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	result, err := h.svc.ListClassDailySummaries(c.Context(), tenantID, schoolID, classID, startDate, endDate)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ── Calendar Status ───────────────────────────────────────────────────────

// GetCalendarStatus handles GET /api/v1/attendance/calendar/status.
func (h *Handler) GetCalendarStatus(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Validate required params
	if startDate == "" || endDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "start_date and end_date are required",
		})
	}

	// Validate date format (basic ISO date check)
	if len(startDate) != 10 || len(endDate) != 10 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "start_date and end_date must be valid ISO dates (YYYY-MM-DD)",
		})
	}

	// Validate end_date >= start_date
	if endDate < startDate {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "end_date must be on or after start_date",
		})
	}

	// Cap range to 62 days (calendar month view — not a bulk export)
	const maxRangeDays = 62
	days := countDays(startDate, endDate)
	if days > maxRangeDays {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "date range must not exceed 62 days",
		})
	}

	result, err := h.svc.GetCalendarStatus(c.Context(), tenantID, schoolID, startDate, endDate)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// countDays returns the number of calendar days between two ISO date strings.
func countDays(startDate, endDate string) int {
	const isoDate = "2006-01-02"
	start, err := time.Parse(isoDate, startDate)
	if err != nil {
		return 0
	}
	end, err := time.Parse(isoDate, endDate)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Hours()/24) + 1
}
