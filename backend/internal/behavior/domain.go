// Package behavior manages school-configurable behavior categories and
// incident/behavior notes that go through admin approval before reaching parents.
package behavior

import (
	"time"

	"somotracker/backend/internal/xerrors"
)

// ─── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = xerrors.NotFound("behavior not found")
	ErrAlreadyExists = xerrors.AlreadyExists("behavior already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid behavior input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("behavior conflict")
	ErrCategoryInUse = xerrors.Conflict("category has active behavior notes")
)

// ─── Types ────────────────────────────────────────────────────────────────

// BehaviorCategory represents a school-configurable incident category.
type BehaviorCategory struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	SchoolID        string    `json:"school_id"`
	Name            string    `json:"name"`
	DefaultSeverity *string   `json:"default_severity,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// CreateCategoryPayload is the request body for creating a behavior category.
type CreateCategoryPayload struct {
	Name            string  `json:"name"`
	DefaultSeverity *string `json:"default_severity,omitempty"`
}

// UpdateCategoryPayload is the request body for updating a behavior category.
type UpdateCategoryPayload struct {
	Name            *string `json:"name,omitempty"`
	DefaultSeverity *string `json:"default_severity,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

// BehaviorNoteStatus represents the lifecycle status of a behavior note.
type BehaviorNoteStatus string

const (
	StatusPendingReview    BehaviorNoteStatus = "PENDING_REVIEW"
	StatusApproved         BehaviorNoteStatus = "APPROVED"
	StatusRejected         BehaviorNoteStatus = "REJECTED"
	StatusIncludedInReport BehaviorNoteStatus = "INCLUDED_IN_REPORT"
)

// BehaviorNote is a single incident/behavior record logged by a teacher.
type BehaviorNote struct {
	ID              string             `json:"id"`
	TenantID        string             `json:"tenant_id"`
	SchoolID        string             `json:"school_id"`
	StudentID       string             `json:"student_id"`
	TimetableSlotID string             `json:"timetable_slot_id"`
	Date            time.Time          `json:"date"`
	CategoryID      string             `json:"category_id"`
	Description     string             `json:"description"`
	IsUrgent        bool               `json:"is_urgent"`
	Status          BehaviorNoteStatus `json:"status"`
	AuthoredByID    string             `json:"authored_by_id"`
	ReviewedByID    *string            `json:"reviewed_by_id,omitempty"`
	ReviewedAt      *time.Time         `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at,omitempty"`
	UpdatedAt       time.Time          `json:"updated_at,omitempty"`
}

// CreateNotePayload is the request body for creating a behavior note.
type CreateNotePayload struct {
	TimetableSlotID string `json:"timetable_slot_id"`
	StudentID       string `json:"student_id"`
	Date            string `json:"date"`
	CategoryID      string `json:"category_id"`
	Description     string `json:"description"`
	IsUrgent        bool   `json:"is_urgent"`
}

// ReviewDecisionPayload is the request body for approving/rejecting a note.
type ReviewDecisionPayload struct {
	Decision  string  `json:"decision"` // "APPROVED" or "REJECTED"
	AdminNote *string `json:"admin_note,omitempty"`
}

// PendingNoteItem is a behavior note in the admin review queue,
// enriched with student/class/category names.
type PendingNoteItem struct {
	ID              string             `json:"id"`
	StudentID       string             `json:"student_id"`
	StudentFullName string             `json:"student_full_name"`
	ClassName       string             `json:"class_name"`
	CategoryID      string             `json:"category_id"`
	CategoryName    string             `json:"category_name"`
	Description     string             `json:"description"`
	IsUrgent        bool               `json:"is_urgent"`
	AuthoredByID    string             `json:"authored_by_id"`
	AuthoredByName  string             `json:"authored_by_name"`
	Date            time.Time          `json:"date"`
	Status          BehaviorNoteStatus `json:"status"`
}

// ── Response Types ───────────────────────────────────────────────────────

// PendingNotesResponse is the admin review queue response.
type PendingNotesResponse struct {
	Notes []PendingNoteItem `json:"notes"`
}

// TeacherNotesResponse is a teacher's own submitted notes.
type TeacherNotesResponse struct {
	Notes []TeacherNoteItem `json:"notes"`
}

// TeacherNoteItem is a note in the teacher's personal list,
// enriched with student name, category name, and review outcome.
type TeacherNoteItem struct {
	ID              string             `json:"id"`
	StudentID       string             `json:"student_id"`
	StudentFullName string             `json:"student_full_name"`
	ClassName       string             `json:"class_name"`
	CategoryName    string             `json:"category_name"`
	Description     string             `json:"description"`
	IsUrgent        bool               `json:"is_urgent"`
	Date            time.Time          `json:"date"`
	Status          BehaviorNoteStatus `json:"status"`
}

// ── Student Behavior Term Summaries ───────────────────────────────────────

// StudentBehaviorTermSummary represents the materialised rollup of behavior
// notes per student per term. Only APPROVED and INCLUDED_IN_REPORT notes
// count toward the main totals (total_incidents, urgent_count, etc.).
// PENDING_REVIEW and REJECTED notes are excluded from report-card totals
// but counted in pending_review_count and resolved_count for admin visibility.
// StudentBehaviorTermSummariesResponse wraps a list of behavior term summaries.
type StudentBehaviorTermSummariesResponse struct {
	Items []StudentBehaviorTermSummary `json:"items"`
	Total int                          `json:"total"`
}

type StudentBehaviorTermSummary struct {
	ID                 string  `json:"id"`
	TenantID           string  `json:"-"`
	SchoolID           string  `json:"-"`
	StudentID          string  `json:"student_id"`
	AcademicTermID     string  `json:"academic_term_id"`
	TotalIncidents     int     `json:"total_incidents"`
	UrgentCount        int     `json:"urgent_count"`
	CommendationsCount int     `json:"commendations_count"`
	DisciplinaryCount  int     `json:"disciplinary_count"`
	PendingReviewCount int     `json:"pending_review_count"`
	ResolvedCount      int     `json:"resolved_count"`
	PrimaryCategoryID  *string `json:"primary_category_id,omitempty"`
	LastRefreshedAt    string  `json:"last_refreshed_at"`
}
