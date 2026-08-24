// Package teacherdeliverysummaries provides incrementally updated summaries of
// teacher lesson delivery metrics per term. Grain: Teacher + Term.
//
// Slot ownership is resolved via timetable_allocations.teacher_id. Summaries are
// updated via batch computation (fn_compute_teacher_delivery_summaries).
package teacherdeliverysummaries

import (
	"context"

	"somotracker/backend/internal/xerrors"
)

// ── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = xerrors.NotFound("teacher delivery summary not found")
	ErrAlreadyExists = xerrors.AlreadyExists("teacher delivery summary already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid teacher delivery summary input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("teacher delivery summary conflict")
)

// ── Domain Models ────────────────────────────────────────────────────────

// TeacherDeliverySummary represents the delivery metrics for a teacher in a term.
type TeacherDeliverySummary struct {
	ID                   string   `json:"id"`
	TenantID             string   `json:"-"`
	SchoolID             string   `json:"-"`
	UserID               string   `json:"user_id"`
	AcademicTermID       string   `json:"academic_term_id"`
	TotalAssignedSlots   int      `json:"total_assigned_slots"`
	MarkedSlots          int      `json:"marked_slots"`
	MissedSlots          int      `json:"missed_slots"`
	SessionsCreated      int      `json:"sessions_created"`
	SessionsApproved     int      `json:"sessions_approved"`
	OnTimeSubmissionRate *float64 `json:"on_time_submission_rate,omitempty"`
	LastRefreshedAt      string   `json:"last_refreshed_at"`
}

// DeliverySummaryListResponse wraps a list of delivery summaries.
type DeliverySummaryListResponse struct {
	Items []TeacherDeliverySummary `json:"items"`
	Total int                      `json:"total"`
}

// TeacherDeliveryBreakdownItem is the per-teacher Marked vs. Missed slot
// rollup for the School Administrator dashboard grouped bar chart. It is a
// read-only view model assembled from teacher_delivery_summaries JOIN users
// (per the CQRS read-model rule — no cross-domain Go imports).
//
// MissedSlots is surfaced first-class because it is the critical metric for
// tracking compliance and skipped roll calls; the endpoint orders by it
// descending so chronic non-compliant teachers appear at the top of the
// chart. TSCNumber is nullable — teachers without a TSC registration number
// still surface with their full name.
type TeacherDeliveryBreakdownItem struct {
	TeacherID            string  `json:"teacher_id"`
	TeacherName          string  `json:"teacher_name"`
	TSCNumber            *string `json:"tsc_number"`
	TotalAssignedSlots   int     `json:"total_assigned_slots"`
	MarkedSlots          int     `json:"marked_slots"`
	MissedSlots          int     `json:"missed_slots"`
	OnTimeSubmissionRate float64 `json:"on_time_submission_rate"`
}

// TeacherDeliveryBreakdownListResponse wraps a list of teacher delivery
// breakdown items.
type TeacherDeliveryBreakdownListResponse struct {
	Items []TeacherDeliveryBreakdownItem `json:"items"`
	Total int                            `json:"total"`
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

// ── Repository Interface ─────────────────────────────────────────────────

// Repository defines the contract for teacher delivery summaries persistence.
type Repository interface {
	// RefreshComputation triggers the batch computation for all teachers in the given term.
	RefreshComputation(ctx context.Context, termID string) error

	// ListByTeacher returns delivery summaries for a specific teacher in a given term.
	ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string) (*DeliverySummaryListResponse, error)

	// ListByTerm returns all delivery summaries for a given term.
	ListByTerm(ctx context.Context, tenantID, schoolID, termID string) (*DeliverySummaryListResponse, error)

	// GetByTeacherTerm returns a single delivery summary for a teacher in a term.
	GetByTeacherTerm(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error)

	// ListDeliveryBreakdown returns per-teacher Marked vs. Missed slot counts
	// for a school in a term, JOINing users for teacher names and TSC numbers.
	// Ordered by missed_slots DESC NULLS LAST so chronic non-compliant
	// teachers surface first.
	ListDeliveryBreakdown(ctx context.Context, tenantID, schoolID, termID string) ([]TeacherDeliveryBreakdownItem, error)
}
