// Package attendance manages per-student, per-timetable-slot attendance records,
// lesson execution sessions, and materialised term summaries.
package attendance

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// ─── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = xerrors.NotFound("attendance not found")
	ErrAlreadyExists = xerrors.AlreadyExists("attendance already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid attendance input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("attendance conflict")
	ErrAlreadyMarked = xerrors.Conflict("attendance already marked for this student and slot")
	ErrBreakSlot     = xerrors.InvalidInput("cannot mark attendance for a break period")
)

// ─── Enums ────────────────────────────────────────────────────────────────

// AttendanceStatus represents a student's attendance state for a single period.
type AttendanceStatus string

const (
	StatusPresent AttendanceStatus = "PRESENT"
	StatusAbsent  AttendanceStatus = "ABSENT"
	StatusLate    AttendanceStatus = "LATE"
	StatusExcused AttendanceStatus = "EXCUSED"
)

// ValidAttendanceStatuses returns all valid attendance status values.
func ValidAttendanceStatuses() []AttendanceStatus {
	return []AttendanceStatus{StatusPresent, StatusAbsent, StatusLate, StatusExcused}
}

// SessionStatus represents the execution status of a lesson session.
type SessionStatus string

const (
	SessionSubmitted SessionStatus = "SUBMITTED"
	SessionSkipped   SessionStatus = "SKIPPED"
)

// ─── Domain Models ────────────────────────────────────────────────────────

// AttendanceRecord is a single student's attendance mark for one timetable slot on one date.
type AttendanceRecord struct {
	ID                    string           `json:"id"`
	TenantID              string           `json:"tenant_id"`
	SchoolID              string           `json:"school_id"`
	StudentID             string           `json:"student_id"`
	TimetableAllocationID string           `json:"timetable_allocation_id"`
	AcademicTermID        string           `json:"academic_term_id"`
	Date                  string           `json:"date"`
	Status                AttendanceStatus `json:"status"`
	MarkedBy              string           `json:"marked_by"`
	Note                  *string          `json:"note,omitempty"`
	AttendanceSessionID   *string          `json:"attendance_session_id,omitempty"`
	CreatedAt             time.Time        `json:"created_at,omitempty"`
	UpdatedAt             time.Time        `json:"updated_at,omitempty"`
}

// AttendanceSession tracks whether a lesson actually took place on a given date.
type AttendanceSession struct {
	ID                    string        `json:"id"`
	TenantID              string        `json:"tenant_id"`
	SchoolID              string        `json:"school_id"`
	TimetableAllocationID string        `json:"timetable_allocation_id"`
	Date                  string        `json:"date"`
	Status                SessionStatus `json:"status"`
	SkipReason            *string       `json:"skip_reason,omitempty"`
	CreatedAt             time.Time     `json:"created_at,omitempty"`
	UpdatedAt             time.Time     `json:"updated_at,omitempty"`
}

// SessionWithEnrichedData extends AttendanceSession with joined data.
type SessionWithEnrichedData struct {
	AttendanceSession
	ClassName      string  `json:"class_name"`
	StreamName     string  `json:"stream_name,omitempty"`
	GradeLevel     string  `json:"grade_level"`
	PeriodName     string  `json:"period_name"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	LearningAreaID string  `json:"learning_area_id,omitempty"`
	LearningArea   *string `json:"learning_area_name,omitempty"`
	TeacherName    *string `json:"teacher_name,omitempty"`
}

// AttendanceTermSummary is a materialised rollup per student × term × learning area.
type AttendanceTermSummary struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	SchoolID             string    `json:"school_id"`
	StudentID            string    `json:"student_id"`
	AcademicTermID       string    `json:"academic_term_id"`
	AcademicYearID       string    `json:"academic_year_id"`
	LearningAreaID       string    `json:"learning_area_id"`
	LearningAreaName     string    `json:"learning_area_name,omitempty"`
	PeriodsTotal         int       `json:"periods_total"`
	PeriodsPresent       int       `json:"periods_present"`
	PeriodsAbsent        int       `json:"periods_absent"`
	PeriodsLate          int       `json:"periods_late"`
	PeriodsExcused       int       `json:"periods_excused"`
	AttendancePercentage float64   `json:"attendance_percentage"`
	LastRefreshedAt      time.Time `json:"last_refreshed_at,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

// ─── Enriched Record (for the marking grid / dashboard view) ──────────────

// RecordWithEnrichedData extends AttendanceRecord with student and slot details.
type RecordWithEnrichedData struct {
	AttendanceRecord
	StudentFullName  string  `json:"student_full_name"`
	ClassName        string  `json:"class_name"`
	GradeLevel       string  `json:"grade_level"`
	StreamName       string  `json:"stream_name,omitempty"`
	PeriodName       string  `json:"period_name"`
	DayOfWeek        int     `json:"day_of_week"`
	StartTime        string  `json:"start_time"`
	EndTime          string  `json:"end_time"`
	LearningAreaID   string  `json:"learning_area_id,omitempty"`
	LearningAreaName *string `json:"learning_area_name,omitempty"`
}

// ─── Payloads ─────────────────────────────────────────────────────────────

// CreateSessionPayload is the request body for creating an attendance session.
type CreateSessionPayload struct {
	TimetableAllocationID string  `json:"timetable_allocation_id"`
	Date                  string  `json:"date"`
	Status                string  `json:"status"` // SUBMITTED or SKIPPED
	SkipReason            *string `json:"skip_reason,omitempty"`
}

// UpdateSessionPayload is the request body for updating a session.
type UpdateSessionPayload struct {
	Status     *string `json:"status,omitempty"`
	SkipReason *string `json:"skip_reason,omitempty"`
}

// StudentAttendanceMark is a single student's mark within a batch.
type StudentAttendanceMark struct {
	StudentID string           `json:"student_id"`
	Status    AttendanceStatus `json:"status"`
	Note      *string          `json:"note,omitempty"`
}

// BatchMarkPayload is the request body for marking attendance in bulk.
type BatchMarkPayload struct {
	Date                  string                  `json:"date"`
	TimetableAllocationID string                  `json:"timetable_allocation_id"`
	Records               []StudentAttendanceMark `json:"records"`
}

// BatchMarkResult holds the outcome of a batch mark operation.
type BatchMarkResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

// UpdateRecordPayload is the request body for updating a single attendance record.
type UpdateRecordPayload struct {
	Status *AttendanceStatus `json:"status,omitempty"`
	Note   *string           `json:"note,omitempty"`
}

// ─── Marked Timetable Allocation Response ───────────────────────────────

type StudentMarkingRecord struct {
	StudentID   string  `json:"student_id"`
	StudentName string  `json:"student_name"`
	Status      string  `json:"status"`
	Note        *string `json:"note,omitempty"`
}

type MarkedTimetableAllocationResponse struct {
	Date           string                 `json:"date"`
	SessionID      *string                `json:"session_id,omitempty"`
	SessionStatus  *string                `json:"session_status,omitempty"`
	SkipReason     *string                `json:"skip_reason,omitempty"`
	ClassID        string                 `json:"class_id"`
	ClassName      string                 `json:"class_name"`
	SubjectID      string                 `json:"subject_id"`
	SubjectName    string                 `json:"subject_name"`
	TeacherID      string                 `json:"teacher_id"`
	TeacherName    string                 `json:"teacher_name"`
	RoomIdentifier *string                `json:"room_identifier,omitempty"`
	Students       []StudentMarkingRecord `json:"students"`
}

// SessionFilter contains optional filter params for listing sessions.
type SessionFilter struct {
	TimetableAllocationID string `json:"timetable_allocation_id,omitempty"`
	Date                  string `json:"date,omitempty"`
	Status                string `json:"status,omitempty"`
	ClassID               string `json:"class_id,omitempty"`
	SchoolID              string `json:"school_id,omitempty"`
	TenantID              string `json:"tenant_id,omitempty"`
}

// RecordFilter contains optional filter params for listing attendance records.
type RecordFilter struct {
	TimetableAllocationID string `json:"timetable_allocation_id,omitempty"`
	Date                  string `json:"date,omitempty"`
	StudentID             string `json:"student_id,omitempty"`
	ClassID               string `json:"class_id,omitempty"`
	AcademicTermID        string `json:"academic_term_id,omitempty"`
	SchoolID              string `json:"school_id,omitempty"`
	TenantID              string `json:"tenant_id,omitempty"`
	Status                string `json:"status,omitempty"`
}

// ─── Calendar Status Types ────────────────────────────────────────────────

// DayStatus represents whether attendance has been fully handled for a day.
type DayStatus string

const (
	DayStatusNone   DayStatus = "none"
	DayStatusGreen  DayStatus = "green"
	DayStatusYellow DayStatus = "yellow"
	DayStatusRed    DayStatus = "red"
)

// CalendarDayStatusRaw is the raw database result (before status mapping).
type CalendarDayStatusRaw struct {
	Date          string `json:"date"`
	ExpectedCount int    `json:"expected_count"`
	HandledCount  int    `json:"handled_count"`
}

// CalendarDayStatus is the per-date attendance completion status with computed status.
type CalendarDayStatus struct {
	Date          string    `json:"date"`
	ExpectedCount int       `json:"expected_count"`
	HandledCount  int       `json:"handled_count"`
	Status        DayStatus `json:"status"`
}

// CalendarStatusListResponse wraps a list of calendar day statuses.
type CalendarStatusListResponse struct {
	Items []CalendarDayStatus `json:"items"`
	Total int                 `json:"total"`
}

// ─── Response Types ───────────────────────────────────────────────────────

// SessionListResponse wraps a list of enriched sessions.
type SessionListResponse struct {
	Items []SessionWithEnrichedData `json:"items"`
	Total int                       `json:"total"`
}

// RecordListResponse wraps a list of enriched attendance records.
type RecordListResponse struct {
	Items []RecordWithEnrichedData `json:"items"`
	Total int                      `json:"total"`
}

// SummaryListResponse wraps a list of attendance term summaries.
type SummaryListResponse struct {
	Items []AttendanceTermSummary `json:"items"`
	Total int                     `json:"total"`
}

// ─── Class Daily Attendance Summary ──────────────────────────────────────

// ClassDailyAttendanceSummary is a materialised rollup per class × date.
type ClassDailyAttendanceSummary struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	SchoolID            string    `json:"school_id"`
	ClassID             string    `json:"class_id"`
	AcademicTermID      string    `json:"academic_term_id"`
	Date                string    `json:"date"`
	TotalEnrolled       int       `json:"total_enrolled"`
	PresentCount        int       `json:"present_count"`
	AbsentCount         int       `json:"absent_count"`
	LateCount           int       `json:"late_count"`
	ExcusedCount        int       `json:"excused_count"`
	DailyAttendanceRate float64   `json:"daily_attendance_rate"`
	LastRefreshedAt     time.Time `json:"last_refreshed_at,omitempty"`
	CreatedAt           time.Time `json:"created_at,omitempty"`
	UpdatedAt           time.Time `json:"updated_at,omitempty"`
}

// ClassDailySummaryListResponse wraps a list of class daily attendance summaries.
type ClassDailySummaryListResponse struct {
	Items []ClassDailyAttendanceSummary `json:"items"`
	Total int                           `json:"total"`
}

// RefreshSummaryResponse is returned after triggering a summary refresh.
type RefreshSummaryResponse struct {
	Message string `json:"message"`
	TermID  string `json:"term_id"`
}

// ─── Class Learning Area Term Summary ──────────────────────────

// ClassLearningAreaTermSummary is a materialised rollup per class × learning area × term.
type ClassLearningAreaTermSummary struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	SchoolID             string    `json:"school_id"`
	ClassID              string    `json:"class_id"`
	LearningAreaID       string    `json:"learning_area_id"`
	AcademicTermID       string    `json:"academic_term_id"`
	AcademicYearID       string    `json:"academic_year_id"`
	StudentsIncluded     int       `json:"students_included"`
	PeriodsTotal         int       `json:"periods_total"`
	PeriodsPresent       int       `json:"periods_present"`
	PeriodsAbsent        int       `json:"periods_absent"`
	PeriodsLate          int       `json:"periods_late"`
	PeriodsExcused       int       `json:"periods_excused"`
	AttendancePercentage float64   `json:"attendance_percentage"`
	LastRefreshedAt      time.Time `json:"last_refreshed_at,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

// ClassLearningAreaTermSummaryListResponse wraps a list of class learning area term summaries.
type ClassLearningAreaTermSummaryListResponse struct {
	Items []ClassLearningAreaTermSummary `json:"items"`
	Total int                            `json:"total"`
}

// ─── Class Term Attendance Summary ─────────────────────────────

// ClassTermAttendanceSummary is a materialised rollup per class × term.
type ClassTermAttendanceSummary struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	SchoolID           string    `json:"school_id"`
	ClassID            string    `json:"class_id"`
	AcademicTermID     string    `json:"academic_term_id"`
	AcademicYearID     string    `json:"academic_year_id"`
	DaysInTerm         int       `json:"days_in_term"`
	TotalEnrolledAvg   float64   `json:"total_enrolled_avg,omitempty"`
	PresentCount       int       `json:"present_count"`
	AbsentCount        int       `json:"absent_count"`
	LateCount          int       `json:"late_count"`
	ExcusedCount       int       `json:"excused_count"`
	TermAttendanceRate float64   `json:"term_attendance_rate"`
	LastRefreshedAt    time.Time `json:"last_refreshed_at,omitempty"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

// ClassTermAttendanceSummaryListResponse wraps a list of class term attendance summaries.
type AttendanceSummaryRow struct {
	ClassName         string  `json:"class_name"`
	TermName          string  `json:"term_name"`
	TermNumber        int     `json:"term_number"`
	AcademicYear      string  `json:"academic_year"`
	PresentPercentage float64 `json:"present_percentage"`
	AbsentPercentage  float64 `json:"absent_percentage"`
	ExcusedPercentage float64 `json:"excused_percentage"`
	LatePercentage    float64 `json:"late_percentage"`
	DaysInTerm        int     `json:"days_in_term"`
	TotalEnrolledAvg  float64 `json:"total_enrolled_avg"`
}

type AttendanceSummaryResponse struct {
	AcademicYear string                 `json:"academic_year"`
	Data         []AttendanceSummaryRow `json:"data"`
}

type ClassTermAttendanceSummaryListResponse struct {
	Items []ClassTermAttendanceSummary `json:"items"`
	Total int                          `json:"total"`
}

// DayOfWeekSummariesResponse represents the response for the day-of-week attendance exceptions endpoint.
type DayOfWeekSummariesResponse struct {
	AcademicYear string                 `json:"academic_year"`
	ClassName    string                 `json:"class_name"`
	Data         []DayOfWeekSummaryItem `json:"data"`
}

// DayOfWeekSummaryItem represents a single day's attendance exceptions.
type DayOfWeekSummaryItem struct {
	DayOfWeekNumber int    `json:"day_of_week_number"`
	DayName         string `json:"day_name"`
	AbsentCount     int    `json:"absent_count"`
	LateCount       int    `json:"late_count"`
	ExcusedCount    int    `json:"excused_count"`
}

// ─── Class Attendance Breakdown (School Admin Dashboard) ──────────────────

// ClassAttendanceBreakdownItem is the per-class Present/Late/Absent rollup for
// the School Administrator dashboard grouped bar chart. It is a read-only view
// model assembled from cbc_classes LEFT JOIN class_term_attendance_summaries
// (per the CQRS read-model rule — no cross-domain Go imports).
//
// AbsentCount is surfaced first-class because it is the critical metric for
// tracking truancy and chronic absenteeism; the endpoint orders by it
// descending so high-absenteeism classes appear at the top of the chart.
type ClassAttendanceBreakdownItem struct {
	ClassID            string  `json:"class_id"`
	ClassName          string  `json:"class_name"`
	TotalEnrolledAvg   float64 `json:"total_enrolled_avg"`
	PresentCount       int     `json:"present_count"`
	LateCount          int     `json:"late_count"`
	AbsentCount        int     `json:"absent_count"`
	ExcusedCount       int     `json:"excused_count"`
	TermAttendanceRate float64 `json:"term_attendance_rate"`
}

// ClassAttendanceBreakdownListResponse wraps a list of class attendance breakdown items.
type ClassAttendanceBreakdownListResponse struct {
	Items []ClassAttendanceBreakdownItem `json:"items"`
	Total int                            `json:"total"`
}

// ─── Learning Area Attendance Breakdown (School Admin Dashboard) ─────────

// LearningAreaAttendanceBreakdownItem is the per-learning-area Present/Absent/
// Excused rollup for the School Administrator dashboard grouped bar chart. It
// is a read-only view model assembled from cbc_learning_areas LEFT JOIN
// class_learning_area_term_summaries aggregated across all classes in the
// school (per the CQRS read-model rule — no cross-domain Go imports).
//
// PeriodsAbsent is surfaced first-class because it is the critical metric for
// tracking truancy and disengagement per subject; the endpoint orders by it
// descending so learning areas with the highest absenteeism appear at the top
// of the chart.
type LearningAreaAttendanceBreakdownItem struct {
	LearningAreaID       string  `json:"learning_area_id"`
	LearningAreaName     string  `json:"learning_area_name"`
	PeriodsTotal         int     `json:"periods_total"`
	PeriodsPresent       int     `json:"periods_present"`
	PeriodsAbsent        int     `json:"periods_absent"`
	PeriodsExcused       int     `json:"periods_excused"`
	AttendancePercentage float64 `json:"attendance_percentage"`
}

// LearningAreaAttendanceBreakdownListResponse wraps a list of learning area
// attendance breakdown items.
type LearningAreaAttendanceBreakdownListResponse struct {
	Items []LearningAreaAttendanceBreakdownItem `json:"items"`
	Total int                                   `json:"total"`
}

// ─── School Attendance KPIs ──────────────────────────────────────────────

// SchoolAttendanceKPI is the macro-level school attendance view model for the
// School Administrator dashboard (School Attendance Command Center). It is a
// read-only rollup assembled from class_daily_attendance_summaries,
// class_term_attendance_summaries, timetable_allocations, and
// cbc_attendance_sessions.
type SchoolAttendanceKPI struct {
	// TodaysAttendanceRate is the average daily attendance rate across all
	// classes on the requested date, from class_daily_attendance_summaries.
	TodaysAttendanceRate float64 `json:"todays_attendance_rate"`

	// TotalPresent is the number of PRESENT marks across all classes on the date.
	TotalPresent int `json:"total_present"`

	// TotalMarkedRecords is the number of marked records (present + absent +
	// late + excused) across all classes on the date.
	TotalMarkedRecords int `json:"total_marked_records"`

	// ActiveTermAttendanceRate is the average term attendance rate across all
	// classes in the active academic term, from class_term_attendance_summaries.
	ActiveTermAttendanceRate float64 `json:"active_term_attendance_rate"`

	// UnmarkedSlotsToday is the count of non-break timetable slots for today
	// that have no attendance session record yet (action required).
	UnmarkedSlotsToday int `json:"unmarked_slots_today"`

	// SkippedSessionsToday is the count of SKIPPED attendance sessions today.
	SkippedSessionsToday int `json:"skipped_sessions_today"`
}

// ClassTermPercentageItem represents the percentage of attendance statuses for a class and term.
type ClassTermPercentageItem struct {
	ClassName         string  `json:"class_name"`
	TermName          string  `json:"term_name"`
	TermNumber        int     `json:"term_number"`
	AcademicYear      string  `json:"academic_year"`
	PresentPercentage float64 `json:"present_percentage"`
	AbsentPercentage  float64 `json:"absent_percentage"`
	ExcusedPercentage float64 `json:"excused_percentage"`
	LatePercentage    float64 `json:"late_percentage"`
}

// ─── Lowest Attendance Students (for ranking) ───────────────────────────────────

// LowestAttendanceStudent represents a student with low attendance for a given period.
type LowestAttendanceStudent struct {
	StudentID            string  `json:"student_id"`
	FirstName            string  `json:"first_name"`
	LastName             string  `json:"last_name"`
	TotalPeriods         int     `json:"total_periods"`
	PresentCount         int     `json:"present_count"`
	AttendancePercentage float64 `json:"attendance_percentage"`
}

// ─── Repository Interface ─────────────────────────────────────────────────

// Repository defines the contract for attendance persistence.
type Repository interface {
	// ── Sessions ──────────────────────────────────────────────────────

	// CreateSession inserts a new attendance session.
	CreateSession(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error)

	// GetSessionByID returns a session by ID.
	GetSessionByID(ctx context.Context, id, tenantID string) (*AttendanceSession, error)

	// GetEnrichedSessionByID returns a session with joined data.
	GetEnrichedSessionByID(ctx context.Context, id, tenantID string) (*SessionWithEnrichedData, error)

	// GetMarkedTimetableAllocation returns all data needed to render the
	// teacher attendance marking view for a single timetable allocation on a date.
	GetMarkedTimetableAllocation(ctx context.Context, tenantID, schoolID, allocationID, academicTermID, date string) (*MarkedTimetableAllocationResponse, error)

	// ListSessions returns all sessions matching the filter (enriched).
	ListSessions(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error)

	// UpdateSession updates a session's status and/or skip reason.
	UpdateSession(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error)

	// GetSessionsForClassDate returns all sessions for a class on a date.
	GetSessionsForClassDate(ctx context.Context, tenantID, schoolID, classID, date string) ([]SessionWithEnrichedData, error)

	// ── Records ───────────────────────────────────────────────────────

	// BatchMark inserts or updates attendance records in bulk (uses UPSERT).
	BatchMark(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy string, termID string) (*BatchMarkResult, error)

	// UpdateRecord updates a single attendance record (status, note).
	UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error)

	// GetRecordByID returns a single attendance record.
	GetRecordByID(ctx context.Context, id, tenantID string) (*AttendanceRecord, error)

	// ListRecordsBySlotDate returns all enriched records for a slot+date.
	ListRecordsBySlotDate(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) ([]RecordWithEnrichedData, error)

	// ListRecordsByStudentTerm returns all enriched records for a student in a term.
	ListRecordsByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]RecordWithEnrichedData, error)

	// ListRecordsByClassDate returns all enriched records for a class on a date.
	ListRecordsByClassDate(ctx context.Context, tenantID, schoolID, classID, date string) ([]RecordWithEnrichedData, error)

	// ListRecords returns records matching the filter.
	ListRecords(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error)

	// GetTermIDByDate returns the academic term ID whose date range covers the
	// given date for this school. Returns ErrInvalidInput if no term is found.
	GetTermIDByDate(ctx context.Context, tenantID, schoolID, date string) (string, error)

	// ── Summaries ─────────────────────────────────────────────────────

	// GetStudentTermSummary returns summaries for a student in a term (all learning areas).
	GetStudentTermSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error)

	// GetClassTermSummary returns summaries for all students in a class for a term.
	GetClassTermSummary(ctx context.Context, tenantID, schoolID, classID, termID string) ([]AttendanceTermSummary, error)

	// RefreshSummaries recomputes the materialised summaries for a term (background task).
	RefreshSummaries(ctx context.Context, tenantID, schoolID, termID string) error

	// ── Class Daily Summaries ───────────────────────────────────────────

	// GetClassDailySummary returns the daily attendance summary for a class on a date.
	GetClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) (*ClassDailyAttendanceSummary, error)

	// RefreshClassDailySummary recomputes the daily summary for a class on a date.
	RefreshClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) error

	// ListClassDailySummaries returns daily summaries for a class within a date range.
	ListClassDailySummaries(ctx context.Context, tenantID, schoolID, classID, startDate, endDate string) ([]ClassDailyAttendanceSummary, error)

	// ── Class Learning Area Term Summaries ─────────────────────────────────

	// GetClassLearningAreaTermSummary returns the class learning area term summary
	// for a class, learning area, and term.
	GetClassLearningAreaTermSummary(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) (*ClassLearningAreaTermSummary, error)

	// ListClassLearningAreaTermSummaries returns all class learning area term summaries
	// for a class and term (or school and term if classID is empty).
	ListClassLearningAreaTermSummaries(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) ([]ClassLearningAreaTermSummary, error)

	// ── Class Term Attendance Summaries ───────────────────────────────────

	// GetClassTermAttendanceSummary returns the class term attendance summary
	// for a class and term.
	GetClassTermAttendanceSummary(ctx context.Context, tenantID, schoolID, classID, termID string) (*ClassTermAttendanceSummary, error)

	// ListClassTermAttendanceSummaries returns all class term attendance summaries
	// for a school and term (or specific class if provided).
	ListClassTermAttendanceSummaries(ctx context.Context, tenantID, schoolID, classID, termID string) ([]ClassTermAttendanceSummary, error)

	// ListAttendanceSummary returns attendance rollup rows (per-class + aggregate)
	// for a school across an entire academic year.
	ListAttendanceSummary(ctx context.Context, tenantID, schoolID, academicYear string) ([]AttendanceSummaryRow, error)

	// ListClassAttendanceBreakdowns returns per-class present/late/absent counts
	// for a school in a term (class names included), ordered by absent count
	// descending so the highest-absenteeism classes surface first.
	ListClassAttendanceBreakdowns(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error)

	// ListLearningAreaBreakdowns returns per-learning-area present/absent/excused
	// period counts for a school in a term, aggregated across all classes
	// (learning area names included), ordered by absent period count descending
	// so the highest-absenteeism subjects surface first (truancy hotspot watch).
	ListLearningAreaBreakdowns(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error)

	// GetDayOfWeekSummaries returns attendance exceptions (absent/late/excused)
	// aggregated by day of week for the current academic year, optionally
	// filtered by a single class. When classID is nil, results are aggregated
	// across all classes in the tenant.
	GetDayOfWeekSummaries(ctx context.Context, tenantID string, classID *string) (DayOfWeekSummariesResponse, error)

	// ── Calendar Status ───────────────────────────────────────────────

	// ListCalendarStatus returns per-date expected/handled slot counts for a
	// school over a date range. Returns one row per date in the range.
	ListCalendarStatus(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error)

	// ── School Attendance KPIs ────────────────────────────────────────────

	// GetSchoolAttendanceKPIs returns macro-level attendance KPIs for a school
	// on a given date: today's attendance rate, active term attendance rate,
	// unmarked timetable slots for the date, and skipped sessions for the date.
	// When termID is empty, the active term containing the date is used; when
	// no term covers the date, the active-term rate degrades to 0.00.
	GetSchoolAttendanceKPIs(ctx context.Context, tenantID, schoolID, date, termID string) (*SchoolAttendanceKPI, error)

	// ListClassTermPercentages returns the percentage of attendance statuses (present, absent,
	// excused, late) for each class and term in the current academic year, with a rollup row
	// for "All" classes.
	ListClassTermPercentages(ctx context.Context, tenantID, schoolID string) ([]ClassTermPercentageItem, error)

	// GetLowestAttendanceStudents returns the N students with the lowest attendance percentage
	// for the current week (or a specified limit). If limit is 0, defaults to 5.
	GetLowestAttendanceStudents(ctx context.Context, tenantID, schoolID string, limit int) ([]LowestAttendanceStudent, error)
}
