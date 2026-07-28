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
	ID                  string           `json:"id"`
	TenantID            string           `json:"tenant_id"`
	SchoolID            string           `json:"school_id"`
	StudentID           string           `json:"student_id"`
	TimetableSlotID     string           `json:"timetable_slot_id"`
	AcademicTermID      string           `json:"academic_term_id"`
	Date                string           `json:"date"`
	Status              AttendanceStatus `json:"status"`
	MarkedBy            string           `json:"marked_by"`
	Note                *string          `json:"note,omitempty"`
	AttendanceSessionID *string          `json:"attendance_session_id,omitempty"`
	CreatedAt           time.Time        `json:"created_at,omitempty"`
	UpdatedAt           time.Time        `json:"updated_at,omitempty"`
}

// AttendanceSession tracks whether a lesson actually took place on a given date.
type AttendanceSession struct {
	ID              string        `json:"id"`
	TenantID        string        `json:"tenant_id"`
	SchoolID        string        `json:"school_id"`
	TimetableSlotID string        `json:"timetable_slot_id"`
	Date            string        `json:"date"`
	Status          SessionStatus `json:"status"`
	SkipReason      *string       `json:"skip_reason,omitempty"`
	CreatedAt       time.Time     `json:"created_at,omitempty"`
	UpdatedAt       time.Time     `json:"updated_at,omitempty"`
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
	TimetableSlotID string  `json:"timetable_slot_id"`
	Date            string  `json:"date"`
	Status          string  `json:"status"` // SUBMITTED or SKIPPED
	SkipReason      *string `json:"skip_reason,omitempty"`
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
	Date            string                  `json:"date"`
	TimetableSlotID string                  `json:"timetable_slot_id"`
	Records         []StudentAttendanceMark `json:"records"`
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

// SessionFilter contains optional filter params for listing sessions.
type SessionFilter struct {
	TimetableSlotID string `json:"timetable_slot_id,omitempty"`
	Date            string `json:"date,omitempty"`
	Status          string `json:"status,omitempty"`
	ClassID         string `json:"class_id,omitempty"`
	SchoolID        string `json:"school_id,omitempty"`
	TenantID        string `json:"tenant_id,omitempty"`
}

// RecordFilter contains optional filter params for listing attendance records.
type RecordFilter struct {
	TimetableSlotID string `json:"timetable_slot_id,omitempty"`
	Date            string `json:"date,omitempty"`
	StudentID       string `json:"student_id,omitempty"`
	ClassID         string `json:"class_id,omitempty"`
	AcademicTermID  string `json:"academic_term_id,omitempty"`
	SchoolID        string `json:"school_id,omitempty"`
	TenantID        string `json:"tenant_id,omitempty"`
	Status          string `json:"status,omitempty"`
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
}
