// Package teacherworkloadsummaries provides batch-computed summaries of teacher
// workload metrics per academic year. Grain: Teacher + Academic Year.
//
// Reassignments via timetable slots or cbc_class_teachers are infrequent, so
// these summaries are computed on-demand rather than incrementally triggered.
package teacherworkloadsummaries

import (
	"context"

	"somotracker/backend/internal/xerrors"
)

// ── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = xerrors.NotFound("teacher workload summary not found")
	ErrAlreadyExists = xerrors.AlreadyExists("teacher workload summary already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid teacher workload summary input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("teacher workload summary conflict")
)

// ── Domain Models ────────────────────────────────────────────────────────

// TeacherWorkloadSummary represents the workload metrics for a teacher in an
// academic year.
type TeacherWorkloadSummary struct {
	ID                    string   `json:"id"`
	TenantID              string   `json:"-"`
	SchoolID              string   `json:"-"`
	UserID                string   `json:"user_id"`
	AcademicYearID        string   `json:"academic_year_id"`
	TotalAssignedPeriods  int      `json:"total_assigned_periods"`
	UniqueSubjects        int      `json:"unique_subjects"`
	ClassesTaught         int      `json:"classes_taught"`
	UtilizationPercentage *float64 `json:"utilization_percentage,omitempty"`
	IsOvercapacity        bool     `json:"is_overcapacity"`
	LastRefreshedAt       string   `json:"last_refreshed_at"`
}

// WorkloadSummaryListResponse wraps a list of workload summaries.
type WorkloadSummaryListResponse struct {
	Items []TeacherWorkloadSummary `json:"items"`
	Total int                      `json:"total"`
}

// RefreshRequest is the request body for triggering a batch refresh.
type RefreshRequest struct {
	AcademicYearID string `json:"academic_year_id"`
}

// RefreshResponse is returned after triggering a batch refresh.
type RefreshResponse struct {
	Message string `json:"message"`
	YearID  string `json:"academic_year_id"`
}

// ── Repository Interface ─────────────────────────────────────────────────

// Repository defines the contract for teacher workload summaries persistence.
type Repository interface {
	// RefreshComputation triggers the batch computation for all teachers in the given academic year.
	RefreshComputation(ctx context.Context, academicYearID string) error

	// ListByTeacher returns workload summaries for a specific teacher in a given year.
	ListByTeacher(ctx context.Context, tenantID, schoolID, userID, yearID string) (*WorkloadSummaryListResponse, error)

	// ListByYear returns all workload summaries for a given academic year.
	ListByYear(ctx context.Context, tenantID, schoolID, yearID string) (*WorkloadSummaryListResponse, error)

	// GetByTeacherYear returns a single workload summary for a teacher in a year.
	GetByTeacherYear(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error)
}
