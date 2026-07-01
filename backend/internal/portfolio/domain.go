package portfolio

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound      = fmt.Errorf("portfolio entry not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("portfolio entry already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid portfolio entry input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("portfolio entry conflict: %w", middleware.ErrConflict)
)

// Module-specific sentinels.
var (
	ErrInvalidEvidenceType = fmt.Errorf("invalid evidence type: %w", middleware.ErrInvalidInput)
	ErrStudentNotFound     = fmt.Errorf("student not found: %w", middleware.ErrNotFound)
	ErrSubStrandNotFound   = fmt.Errorf("sub-strand not found: %w", middleware.ErrNotFound)
	ErrDuplicateAdvised    = fmt.Errorf("entry with same student, sub_strand, and evidence_type already exists: %w", middleware.ErrConflict)
)

// ============================================================================
// Domain Models
// ============================================================================

// PortfolioEntry represents a single piece of learner evidence attached to
// a CBC sub-strand. One entry per (student, sub_strand, evidence_type)
// is advised but not enforced at the DB level.
type PortfolioEntry struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"-"`
	StudentID      string  `json:"student_id"`
	SubStrandID    string  `json:"sub_strand_id"`
	EvidenceType   string  `json:"evidence_type"`
	StoragePointer string  `json:"storage_pointer"`
	LinkedResultID *string `json:"linked_result_id,omitempty"`
	DateCollected  *string `json:"date_collected,omitempty"` // "YYYY-MM-DD"
	CreatedAt      string  `json:"created_at"`
}

// ============================================================================
// Valid Evidence Types
// ============================================================================

var validEvidenceTypes = map[string]bool{
	"Physical_File_Reference": true,
	"Digital_Artifact_URL":    true,
	"Video_Recording":         true,
	"Audio_Log":               true,
	"Observation_Checklist":   true,
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

type CreatePortfolioEntryPayload struct {
	StudentID      string  `json:"student_id"`
	SubStrandID    string  `json:"sub_strand_id"`
	EvidenceType   string  `json:"evidence_type"`
	StoragePointer string  `json:"storage_pointer"`
	LinkedResultID *string `json:"linked_result_id,omitempty"`
	DateCollected  *string `json:"date_collected,omitempty"`
}

type UpdatePortfolioEntryPayload struct {
	StoragePointer *string `json:"storage_pointer,omitempty"`
	DateCollected  *string `json:"date_collected,omitempty"`
	LinkedResultID *string `json:"linked_result_id,omitempty"` // set to "" to unlink
}

type ListEntriesQuery struct {
	StudentID   string
	SubStrandID string
}

type CreateEntryResponse struct {
	ID string `json:"id"`
}

type ListEntriesResponse struct {
	Data []PortfolioEntry `json:"data"`
}

type EntryResponse struct {
	Data PortfolioEntry `json:"data"`
}

// ============================================================================
// Cross-Domain Resolver Interfaces
// ============================================================================

// StudentResolver validates that a student exists within a tenant.
// Implemented by the students repository and wired via fx in main.go.
type StudentResolver interface {
	StudentExists(ctx context.Context, tenantID, studentID string) (bool, error)
}

// SubStrandResolver validates that a sub-strand exists within a tenant/school.
// Implemented by the curriculum repository and wired via fx in main.go.
type SubStrandResolver interface {
	SubStrandExists(ctx context.Context, subStrandID string) (bool, error)
}

// ============================================================================
// Repository Interface
// ============================================================================

// Repository defines the contract for portfolio entry persistence.
type Repository interface {
	CreateEntry(ctx context.Context, e *PortfolioEntry) (string, error)
	GetEntryByID(ctx context.Context, id, tenantID string) (*PortfolioEntry, error)
	ListEntries(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error)
	UpdateEntry(ctx context.Context, e *PortfolioEntry) error
	DeleteEntry(ctx context.Context, id, tenantID string) error

	// Duplicate check — advisory, not enforced by unique constraint.
	EntryExistsForStudentSubStrandEvidence(ctx context.Context, tenantID, studentID, subStrandID, evidenceType string) (bool, error)
}
