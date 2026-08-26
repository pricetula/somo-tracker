package attendance

import (
	"somotracker/backend/internal/academicyears"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
	"somotracker/backend/internal/xerrors"
)

// Handler exposes attendance HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc academicyears.AcademicYearTermResolver
}

// NewHandler creates a new attendance Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicyears.AcademicYearTermResolver) {
	h.academicYearsSvc = aySvc
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

	// Class learning area term summaries
	classLA := router.Group("/api/v1/attendance/class-learning-area")
	classLA.Get("/breakdown", middleware.RequireAuth, h.ListLearningAreaBreakdowns)
	classLA.Get("/class/:class_id/term/:term_id", middleware.RequireAuth, h.ListClassLearningAreaTermSummaries)
	classLA.Get("/class/:class_id/learning-area/:learning_area_id/term/:term_id", middleware.RequireAuth, h.GetClassLearningAreaTermSummary)
	classLA.Post("/class/:class_id/term/:term_id/refresh", middleware.RequireAuth, h.RefreshClassLearningAreaTermSummary)

	// Class term attendance summaries
	classTerm := router.Group("/api/v1/attendance/class-term")
	classTerm.Get("/breakdown", middleware.RequireAuth, h.ListClassAttendanceBreakdowns)
	classTerm.Get("/class/:class_id/term/:term_id", middleware.RequireAuth, h.GetClassTermAttendanceSummary)
	classTerm.Get("/term/:term_id", middleware.RequireAuth, h.ListClassTermAttendanceSummaries)
	classTerm.Post("/class/:class_id/term/:term_id/refresh", middleware.RequireAuth, h.RefreshClassTermAttendanceSummary)

	// Calendar status (monthly overview)
	calendar := router.Group("/api/v1/attendance/calendar")
	calendar.Get("/status", middleware.RequireAuth, h.GetCalendarStatus)

	// Day-of-week attendance exceptions (weekday stacked bar chart)
	router.Group("/api/v1/attendance").Get("/day-of-week-summaries", middleware.RequireAuth, h.GetDayOfWeekSummaries)

	// School attendance KPIs (School Administrator dashboard)
	// kpis := router.Group("/api/v1/attendance/kpis")
	// Students attendance rankings
	students := router.Group("/api/v1/attendance/students")
	students.Get("/lowest-attendance", middleware.RequireAuth, h.GetLowestAttendanceStudents)
	router.Group("/api/v1/attendance").Get("/class-term-percentages", middleware.RequireAuth, h.GetClassTermPercentages)
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

// resolveCurrentTerm resolves the current academic term ID for the school.
// Returns empty string if no current term is set.
func (h *Handler) resolveCurrentTerm(c *fiber.Ctx, tenantID, schoolID string) (string, error) {
	if h.academicYearsSvc == nil {
		return "", nil
	}
	_, termID, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	return termID, err
}

// ── Sessions ──────────────────────────────────────────────────────────────

// CreateSession handles POST /api/v1/attendance/sessions.
// Academic term is derived from the date when marking records (BatchMark).
func (h *Handler) CreateSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	var payload CreateSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("invalid request body"))
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
		TimetableAllocationID: c.Query("timetable_allocation_id"),
		Date:                  c.Query("date"),
		Status:                c.Query("status"),
		ClassID:               c.Query("class_id"),
		SchoolID:              schoolID,
		TenantID:              tenantID,
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
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("invalid request body"))
	}

	session, err := h.svc.UpdateSession(c.Context(), id, tenantID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(session)
}

// ── Records ───────────────────────────────────────────────────────────────

// BatchMark handles POST /api/v1/attendance/records/batch.
// Academic term is resolved server-side from the current active term if not provided.
func (h *Handler) BatchMark(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return middleware.HTTPError(c, xerrors.Unauthorized("user not authenticated"))
	}

	var payload BatchMarkPayload
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("invalid request body"))
	}

	// Resolve academic term: use query param if provided, otherwise resolve current term
	termID := c.Query("term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

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

	timetableSlotID := c.Query("timetable_allocation_id")
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
		TimetableAllocationID: c.Query("timetable_allocation_id"),
		Date:                  c.Query("date"),
		StudentID:             c.Query("student_id"),
		ClassID:               c.Query("class_id"),
		AcademicTermID:        c.Query("academic_term_id"),
		Status:                c.Query("status"),
		SchoolID:              schoolID,
		TenantID:              tenantID,
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
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("invalid request body"))
	}

	record, err := h.svc.UpdateRecord(c.Context(), id, tenantID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(record)
}

// ── Summaries ─────────────────────────────────────────────────────────────

// GetStudentTermSummary handles GET /api/v1/attendance/summaries/student/:student_id.
// Academic term is resolved server-side from the current active term if not provided.
func (h *Handler) GetStudentTermSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	studentID := c.Params("student_id")
	termID := c.Query("term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	result, err := h.svc.GetStudentTermSummary(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetClassTermSummary handles GET /api/v1/attendance/summaries/class/:class_id.
// Academic term is resolved server-side from the current active term if not provided.
func (h *Handler) GetClassTermSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Params("class_id")
	termID := c.Query("term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	result, err := h.svc.GetClassTermSummary(c.Context(), tenantID, schoolID, classID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// RefreshSummaries handles POST /api/v1/attendance/summaries/refresh.
// Academic term is resolved server-side from the current active term if not provided.
func (h *Handler) RefreshSummaries(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	var payload struct {
		TermID string `json:"term_id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.HTTPError(c, xerrors.UnprocessableEntity("invalid request body"))
	}

	// Resolve term if not provided
	termID := payload.TermID
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	result, err := h.svc.RefreshSummaries(c.Context(), tenantID, schoolID, termID)
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

// ── Class Learning Area Term Summaries ───────────────────────────────────

// GetClassLearningAreaTermSummary handles GET /api/v1/attendance/class-learning-area/class/:class_id/learning-area/:learning_area_id/term/:term_id.
func (h *Handler) GetClassLearningAreaTermSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.GetClassLearningAreaTermSummary(c.Context(), tenantID, schoolID,
		c.Params("class_id"), c.Params("learning_area_id"), c.Params("term_id"))
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListClassLearningAreaTermSummaries handles GET /api/v1/attendance/class-learning-area/class/:class_id/term/:term_id.
func (h *Handler) ListClassLearningAreaTermSummaries(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.ListClassLearningAreaTermSummaries(c.Context(), tenantID, schoolID,
		c.Params("class_id"), c.Query("learning_area_id"), c.Params("term_id"))
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// RefreshClassLearningAreaTermSummary handles POST /api/v1/attendance/class-learning-area/class/:class_id/term/:term_id/refresh.
func (h *Handler) RefreshClassLearningAreaTermSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.RefreshClassLearningAreaTermSummary(c.Context(), tenantID, schoolID,
		c.Params("term_id"), c.Params("class_id"))
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ── Class Term Attendance Summaries ───────────────────────────────────────

// GetClassTermAttendanceSummary handles GET /api/v1/attendance/class-term/class/:class_id/term/:term_id.
func (h *Handler) GetClassTermAttendanceSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.GetClassTermAttendanceSummary(c.Context(), tenantID, schoolID,
		c.Params("class_id"), c.Params("term_id"))
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListClassTermAttendanceSummaries handles GET /api/v1/attendance/class-term/term/:term_id.
func (h *Handler) ListClassTermAttendanceSummaries(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.ListClassTermAttendanceSummaries(c.Context(), tenantID, schoolID,
		c.Query("class_id"), c.Params("term_id"))
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListClassAttendanceBreakdowns handles GET /api/v1/attendance/class-term/breakdown.
//
// Query params:
//   - academic_term_id (UUID, optional) — the term to aggregate
//     (class_term_attendance_summaries are per class × term).
//     If not provided, the current active term is used.
//
// tenant_id and school_id are resolved from the authenticated local context.
// Returns per-class Present/Late/Absent counts ordered by absent_count
// descending so high-absenteeism classes surface first.
func (h *Handler) ListClassAttendanceBreakdowns(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	termID := c.Query("academic_term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	result, err := h.svc.ListClassAttendanceBreakdowns(c.Context(), tenantID, schoolID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListLearningAreaBreakdowns handles GET /api/v1/attendance/class-learning-area/breakdown.
//
// Query params:
//   - academic_term_id (UUID, optional) — the term to aggregate
//     (class_learning_area_term_summaries are per class × learning area × term).
//     If not provided, the current active term is used.
//
// tenant_id and school_id are resolved from the authenticated local context.
// Returns per-learning-area Present/Absent/Excused period counts aggregated
// across all classes, ordered by periods_absent descending so the
// highest-absenteeism subjects surface first (truancy hotspot watch).
func (h *Handler) ListLearningAreaBreakdowns(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	termID := c.Query("academic_term_id")
	if termID == "" {
		termID, err = h.resolveCurrentTerm(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if termID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_TERM",
				"message": "No current academic term is active.",
			})
		}
	}

	result, err := h.svc.ListLearningAreaBreakdowns(c.Context(), tenantID, schoolID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetDayOfWeekSummaries handles GET /api/v1/attendance/day-of-week-summaries.
//
// Query params:
//   - class_id (UUID, optional) — when provided, results are scoped to a single
//     class; when omitted, results are aggregated across all classes.
//
// tenant_id is resolved from the authenticated local context. Returns
// absent/late/excused counts aggregated by day of week (Monday–Friday) for the
// current academic year, ordered by day of week ascending.
func (h *Handler) GetDayOfWeekSummaries(c *fiber.Ctx) error {
	tenantID, _, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	classID := c.Query("class_id")
	var classIDPtr *string
	if classID != "" {
		classIDPtr = &classID
	}

	result, err := h.svc.GetDayOfWeekSummaries(c.Context(), tenantID, classIDPtr)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// RefreshClassTermAttendanceSummary handles POST /api/v1/attendance/class-term/class/:class_id/term/:term_id/refresh.
func (h *Handler) RefreshClassTermAttendanceSummary(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.RefreshClassTermAttendanceSummary(c.Context(), tenantID, schoolID,
		c.Params("term_id"), c.Params("class_id"))
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

// ── School Attendance KPIs ────────────────────────────────────────────────

// GetSchoolAttendanceKPIs handles GET /api/v1/attendance/kpis/school.
func (h *Handler) GetSchoolAttendanceKPIs(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	date := c.Query("date")
	termID := c.Query("term_id")

	if date == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "date is required (YYYY-MM-DD)",
		})
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "date must be a valid ISO date (YYYY-MM-DD)",
		})
	}

	kpi, err := h.svc.GetSchoolAttendanceKPIs(c.Context(), tenantID, schoolID, date, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(kpi)
}

// GetClassTermPercentages handles GET /api/v1/attendance/class-term-percentages.
func (h *Handler) GetClassTermPercentages(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.ListClassTermPercentages(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"academic_year": result[0].AcademicYear,
		"data":          result,
	})
}

// GetLowestAttendanceStudents handles GET /api/v1/attendance/students/lowest-attendance.
func (h *Handler) GetLowestAttendanceStudents(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.attMiddleware(c)
	if err != nil {
		return err
	}

	limitStr := c.Query("limit")
	limit := 5
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	result, err := h.svc.GetLowestAttendanceStudents(c.Context(), tenantID, schoolID, limit)
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
