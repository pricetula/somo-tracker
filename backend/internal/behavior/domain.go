// Package behavior manages school-configurable behavior categories and
// incident/behavior notes that go through admin approval before reaching parents.
package behavior

import (
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// ─── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound      = fmt.Errorf("behavior not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("behavior already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid behavior input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("behavior conflict: %w", middleware.ErrConflict)
	ErrCategoryInUse = fmt.Errorf("category has active behavior notes: %w", middleware.ErrConflict)
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

// PendingNotesResponse is the admin review queue response.
type PendingNotesResponse struct {
	Notes []PendingNoteItem `json:"notes"`
}
