// Package attendance manages per-period attendance records tied to the
// existing timetable structure. One row per student, per timetable slot
// occurrence, per date.
package attendance

import (
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// ─── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound          = fmt.Errorf("attendance not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists     = fmt.Errorf("attendance already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput      = fmt.Errorf("invalid attendance input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized      = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden         = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict          = fmt.Errorf("attendance conflict: %w", middleware.ErrConflict)
	ErrSlotAlreadyMarked = fmt.Errorf("timetable slot already has attendance records for this date: %w", middleware.ErrConflict)
)

// ─── Types ────────────────────────────────────────────────────────────────

// AttendanceStatus represents the possible attendance states for a period.
type AttendanceStatus string

const (
	StatusPresent AttendanceStatus = "PRESENT"
	StatusAbsent  AttendanceStatus = "ABSENT"
	StatusLate    AttendanceStatus = "LATE"
	StatusExcused AttendanceStatus = "EXCUSED"
)

// AttendanceRecord is a single attendance entry per student, per slot, per date.
type AttendanceRecord struct {
	ID              string           `json:"id"`
	TenantID        string           `json:"tenant_id"`
	SchoolID        string           `json:"school_id"`
	StudentID       string           `json:"student_id"`
	TimetableSlotID string           `json:"timetable_slot_id"`
	AcademicTermID  string           `json:"academic_term_id"`
	Date            string           `json:"date"`
	Status          AttendanceStatus `json:"status"`
	MarkedBy        string           `json:"marked_by"`
	MarkedAt        time.Time        `json:"marked_at"`
	Note            *string          `json:"note,omitempty"`
	CreatedAt       time.Time        `json:"created_at,omitempty"`
	UpdatedAt       time.Time        `json:"updated_at,omitempty"`
}

// RosterStudent represents a single student on a class roster for a given slot.
type RosterStudent struct {
	StudentID       string            `json:"student_id"`
	FullName        string            `json:"full_name"`
	AdmissionNumber string            `json:"admission_number"`
	CurrentStatus   *AttendanceStatus `json:"current_status,omitempty"`
}

// SlotRosterResponse is the response for GET roster for a slot.
type SlotRosterResponse struct {
	TimetableSlotID string          `json:"timetable_slot_id"`
	ClassName       string          `json:"class_name"`
	LearningArea    string          `json:"learning_area"`
	Date            string          `json:"date"`
	Students        []RosterStudent `json:"students"`
}

// BulkAttendanceEntry is a single entry in a bulk attendance submission.
type BulkAttendanceEntry struct {
	StudentID string           `json:"student_id"`
	Status    AttendanceStatus `json:"status"`
	Note      *string          `json:"note,omitempty"`
}

// BulkAttendancePayload is the request body for marking attendance in bulk.
type BulkAttendancePayload struct {
	TimetableSlotID string                `json:"timetable_slot_id"`
	Date            string                `json:"date"`
	Entries         []BulkAttendanceEntry `json:"entries"`
}

// StudentAttendanceRecord enriches AttendanceRecord with subject/class info
// for the parent-facing summary view.
type StudentAttendanceRecord struct {
	Date    string           `json:"date"`
	Subject string           `json:"subject"`
	Status  AttendanceStatus `json:"status"`
}

// ChildAttendanceSummary is the response for a parent viewing their child's attendance.
type ChildAttendanceSummary struct {
	StudentID            string                    `json:"student_id"`
	TermID               string                    `json:"term_id"`
	AttendancePercentage float64                   `json:"attendance_percentage"`
	RecentPeriods        []StudentAttendanceRecord `json:"recent_periods"`
}

// AttendanceTermSummary is the materialised rollup row.
type AttendanceTermSummary struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	SchoolID             string    `json:"school_id"`
	StudentID            string    `json:"student_id"`
	AcademicTermID       string    `json:"academic_term_id"`
	LearningAreaID       string    `json:"learning_area_id"`
	PeriodsTotal         int       `json:"periods_total"`
	PeriodsPresent       int       `json:"periods_present"`
	PeriodsAbsent        int       `json:"periods_absent"`
	PeriodsLate          int       `json:"periods_late"`
	PeriodsExcused       int       `json:"periods_excused"`
	AttendancePercentage float64   `json:"attendance_percentage"`
	LastRefreshedAt      time.Time `json:"last_refreshed_at"`
}

// ─── Asynq task types ───────────────────────────────────────────────

// TypeRecomputeClassSummaries is the Asynq task type for recomputing
// attendance_term_summaries for a single class within a term.
const TypeRecomputeClassSummaries = "attendance:recompute_class_summaries"

// recomputeClassPayload is the Asynq task payload for a class-scoped
// attendancesummary recompute.
type recomputeClassPayload struct {
	TenantID string `json:"tenant_id"`
	SchoolID string `json:"school_id"`
	TermID   string `json:"term_id"`
	ClassID  string `json:"class_id"`
}

// UpdateAttendanceEntryPayload is for a single record correction (admin).
type UpdateAttendanceEntryPayload struct {
	Status AttendanceStatus `json:"status"`
	Note   *string          `json:"note,omitempty"`
}

// CompletionStatus represents marking progress for a class on a given day.
type CompletionStatus struct {
	ClassID        string `json:"class_id"`
	ClassName      string `json:"class_name"`
	SlotID         string `json:"slot_id"`
	PeriodName     string `json:"period_name"`
	LearningArea   string `json:"learning_area"`
	LearningAreaID string `json:"learning_area_id"`
	TotalSlots     int    `json:"total_slots"`
	MarkedSlots    int    `json:"marked_slots"`
	IsComplete     bool   `json:"is_complete"`
}

// DashboardFilter holds optional filters for the admin dashboard listing.
type DashboardFilter struct {
	EducationLevels []string
	GradeLevels     []string
	ClassID         string
	IsComplete      string // "complete", "incomplete", or "" (all)
	Page            int
	Limit           int
}

// AdminDashboardResponse is the paginated school-wide attendance dashboard for admins.
type AdminDashboardResponse struct {
	Date  string             `json:"date"`
	Items []CompletionStatus `json:"items"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// StudentHistoryFilter are query params for filtering a student's attendance history.
type StudentHistoryFilter struct {
	TermID         string `json:"term_id,omitempty"`
	StartDate      string `json:"start_date,omitempty"`
	EndDate        string `json:"end_date,omitempty"`
	LearningAreaID string `json:"learning_area_id,omitempty"`
}
