// Package cohortpositions implements periodic batch computation of class and
// grade rankings for all students in a given academic term.
//
// This package is a periodic-only module. Rankings are NEVER updated
// incrementally — computing a single student's rank requires knowing every
// other student's score in the same class and grade for that term. An
// incremental trigger would mean recomputing the whole cohort on every
// single score entry, which doesn't scale during grading season.
//
// The batch function fn_compute_cohort_positions_for_term() is triggered on
// a schedule (e.g. every 30 minutes during active grading windows, then
// locked once the term closes).
package cohortpositions

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// ─── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = fmt.Errorf("cohort position not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("cohort position already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid cohort position input: %w", middleware.ErrInvalidInput)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
)

// ─── Domain Model ─────────────────────────────────────────────────────────

// CohortPositionSummary is a single row in the student_cohort_position_summaries
// table. It captures a student's rank, percentile, and comparative metrics
// within their class and grade for a given academic term.
type CohortPositionSummary struct {
	ID                    string    `json:"id"`
	TenantID              string    `json:"tenant_id"`
	SchoolID              string    `json:"school_id"`
	StudentID             string    `json:"student_id"`
	ClassID               string    `json:"class_id"`
	AcademicTermID        string    `json:"academic_term_id"`
	StudentScore          *float64  `json:"student_score,omitempty"`
	ClassRank             *int      `json:"class_rank,omitempty"`
	ClassHeadcount        *int      `json:"class_headcount,omitempty"`
	ClassAverage          *float64  `json:"class_average,omitempty"`
	ClassPercentile       *float64  `json:"class_percentile,omitempty"`
	GradeRank             *int      `json:"grade_rank,omitempty"`
	GradeHeadcount        *int      `json:"grade_headcount,omitempty"`
	GradeAverage          *float64  `json:"grade_average,omitempty"`
	GradePercentile       *float64  `json:"grade_percentile,omitempty"`
	VarianceFromGradeMean *float64  `json:"variance_from_grade_mean,omitempty"`
	LastRefreshedAt       time.Time `json:"last_refreshed_at"`
	CreatedAt             time.Time `json:"created_at,omitempty"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
}

// ─── Request / Response Payloads ─────────────────────────────────────────

// RefreshRequest is the request body for triggering a cohort position refresh.
type RefreshRequest struct {
	AcademicTermID string `json:"academic_term_id"`
}

// RefreshResponse is returned after triggering a batch refresh.
type RefreshResponse struct {
	Message string `json:"message"`
	TermID  string `json:"term_id"`
}

// BulkPositionResponse wraps a list of cohort position summaries.
type BulkPositionResponse struct {
	Items []CohortPositionSummary `json:"items"`
	Total int                     `json:"total"`
}

// ─── Repository Interface ─────────────────────────────────────────────────

// Repository defines the contract for cohort position persistence.
type Repository interface {
	// RefreshTerm recomputes cohort positions for all students in the given
	// academic term by calling fn_compute_cohort_positions_for_term().
	RefreshTerm(ctx context.Context, termID string) error

	// GetByStudentTerm returns the cohort position for a specific student
	// in a specific term.
	GetByStudentTerm(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error)

	// ListByClassTerm returns all cohort positions for a class in a term.
	ListByClassTerm(ctx context.Context, classID, termID, tenantID string) ([]CohortPositionSummary, error)

	// ListByGradeTerm returns all cohort positions for all classes at the same
	// grade level in a term (across the school).
	ListByGradeTerm(ctx context.Context, schoolID, gradeLevel, termID, tenantID string) ([]CohortPositionSummary, error)
}
