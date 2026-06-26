package attendance

import (
	"context"
	"errors"
	"time"

	"somotracker/backend/internal/timetable"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("attendance not found")
	ErrAlreadyExists = errors.New("attendance already exists")
	ErrInvalidInput  = errors.New("invalid attendance input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("attendance conflict")
)

// AttendanceStatus mirrors the PostgreSQL attendance_status enum.
type AttendanceStatus string

const (
	StatusPresent AttendanceStatus = "PRESENT"
	StatusAbsent  AttendanceStatus = "ABSENT"
	StatusLate    AttendanceStatus = "LATE"
	StatusExcused AttendanceStatus = "EXCUSED"
)

// AttendancePeriod represents a row in cbc_attendance_periods.
type AttendancePeriod struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	SchoolID         string                 `json:"school_id"`
	AcademicTermID   string                 `json:"academic_term_id"`
	ClassID          string                 `json:"class_id"`
	LearningAreaID   string                 `json:"learning_area_id"`
	DateRecorded     string                 `json:"date_recorded"`
	RecordedBy       string                 `json:"recorded_by"`
	AuthorizedByRole *timetable.TeacherRole `json:"authorized_by_role,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// AttendanceLog represents a row in cbc_attendance_logs.
type AttendanceLog struct {
	ID         string           `json:"id"`
	TenantID   string           `json:"tenant_id"`
	PeriodID   string           `json:"attendance_period_id"`
	StudentID  string           `json:"student_id"`
	Status     AttendanceStatus `json:"status"`
	Remarks    *string          `json:"remarks,omitempty"`
	RecordedBy string           `json:"recorded_by"`
}

// AttendanceLogInput is the input for a single attendance log entry.
type AttendanceLogInput struct {
	StudentID string           `json:"student_id"`
	Status    AttendanceStatus `json:"status"`
	Remarks   *string          `json:"remarks,omitempty"`
}

// MarkAttendanceInput is the full batch input for opening/submitting attendance.
type MarkAttendanceInput struct {
	SchoolID       string               `json:"-"`
	AcademicTermID string               `json:"academic_term_id"`
	ClassID        string               `json:"class_id"`
	LearningAreaID string               `json:"learning_area_id"`
	Date           string               `json:"date"`
	PeriodID       *string              `json:"period_id,omitempty"`
	Students       []AttendanceLogInput `json:"students"`
}

// OpenPeriodInput is the input for finding or creating an attendance period.
type OpenPeriodInput struct {
	TenantID       string `json:"-"`
	SchoolID       string `json:"-"`
	AcademicTermID string `json:"academic_term_id"`
	ClassID        string `json:"class_id"`
	LearningAreaID string `json:"learning_area_id"`
	Date           string `json:"date"`
	RecordedBy     string `json:"-"`
}

// AuthorizedRecorderResult holds the result of an authorization check.
type AuthorizedRecorderResult struct {
	Authorized bool                   `json:"authorized"`
	Role       *timetable.TeacherRole `json:"role,omitempty"`
}

// Repository defines the contract for attendance persistence.
type Repository interface {
	GetOrCreatePeriod(ctx context.Context, input OpenPeriodInput) (*AttendancePeriod, bool, error)
	UpdatePeriodAuthorizedBy(ctx context.Context, periodID, tenantID string, role timetable.TeacherRole) error
	UpsertAttendanceLogs(ctx context.Context, tenantID, periodID, recordedBy string, logs []AttendanceLogInput) error
	GetPeriodLogs(ctx context.Context, tenantID, periodID string) (*AttendancePeriod, []AttendanceLog, error)
	IsAuthorizedRecorder(ctx context.Context, tenantID, userID, classID, learningAreaID, termID string) (*AuthorizedRecorderResult, error)
}
