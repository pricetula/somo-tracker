// Package teacherperformance computes periodic batch summaries of teacher
// effectiveness per learning area, class, and term.
//
// Teacher attribution: resolved from cbc_class_teachers at computation time
// via the current SUBJECT_TEACHER row. There is no historical assignment
// tracking, so a mid-term substitute or reassignment gets folded into whoever
// holds the role at computation time. This is an approximation — flag it in
// the UI as such rather than hiding it.
package teacherperformance

import (
	"somotracker/backend/internal/xerrors"
)

// ── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = xerrors.NotFound("teacher performance not found")
	ErrAlreadyExists = xerrors.AlreadyExists("teacher performance already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid teacher performance input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("teacher performance conflict")
)

// ── Domain Models ────────────────────────────────────────────────────────

// TeacherSubjectPerformanceSummary represents the periodic batch-computed
// teacher effectiveness summary per subject per class per term.
type TeacherSubjectPerformanceSummary struct {
	ID                      string   `json:"id"`
	TenantID                string   `json:"-"`
	SchoolID                string   `json:"-"`
	UserID                  string   `json:"user_id"`
	LearningAreaID          string   `json:"learning_area_id"`
	ClassID                 string   `json:"class_id"`
	AcademicTermID          string   `json:"academic_term_id"`
	SubjectMeanScore        *float64 `json:"subject_mean_score,omitempty"`
	CohortMasteryRate       *float64 `json:"cohort_mastery_rate,omitempty"`
	StudentGrowthRate       *float64 `json:"student_growth_rate,omitempty"`
	AssessmentTimelinessIdx *float64 `json:"assessment_timeliness_index,omitempty"`
	StrandCoverageRate      *float64 `json:"strand_coverage_rate,omitempty"`
	LastRefreshedAt         string   `json:"last_refreshed_at"`
}

// TeacherPerformanceListResponse wraps a list of teacher performance summaries.
type TeacherPerformanceListResponse struct {
	Items []TeacherSubjectPerformanceSummary `json:"items"`
	Total int                                `json:"total"`
}

// RefreshRequest is the request body for triggering a batch refresh.
type RefreshRequest struct {
	AcademicTermID string `json:"academic_term_id"`
}

// RefreshResponse is returned after triggering a batch refresh.
type RefreshResponse struct {
	Message string `json:"message"`
	TermID  string `json:"term_id"`
}
