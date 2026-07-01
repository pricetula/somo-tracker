package summaries

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound      = fmt.Errorf("summary not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("summary already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid summary input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("summary conflict: %w", middleware.ErrConflict)
)

// Module-specific sentinels.
var (
	ErrAlreadySynced      = fmt.Errorf("summary already synced: %w", middleware.ErrConflict)
	ErrInvalidRubricLevel = fmt.Errorf("invalid rubric level: %w", middleware.ErrInvalidInput)
)

// ============================================================================
// Domain Models
// ============================================================================

// CompetencySummary represents the definitive per-term competency record
// per learner per learning area.
type CompetencySummary struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"-"`
	StudentID       string  `json:"student_id"`
	LearningAreaID  string  `json:"learning_area_id"`
	ClassID         string  `json:"class_id"`
	AcademicYear    int     `json:"academic_year"`
	Term            int     `json:"term"`
	CalculatedLevel string  `json:"calculated_level"`         // cbc_rubric_level_with_sub_levels
	OverrideLevel   *string `json:"override_level,omitempty"` // cbc_rubric_level_with_sub_levels
	FinalLevel      string  `json:"final_level"`              // cbc_rubric_level (base 4-level)
	KNECSyncStatus  string  `json:"knec_sync_status"`         // Pending | Synced | Failed
	KNECSyncedAt    *string `json:"knec_synced_at,omitempty"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

type ListSummariesQuery struct {
	StudentID      string
	ClassID        string
	LearningAreaID string
	AcademicYear   int
	Term           int
}

type ListSummariesResponse struct {
	Data []CompetencySummary `json:"data"`
}

type CompetencySummaryDetailResponse struct {
	Data CompetencySummary `json:"data"`
}

type OverrideLevelPayload struct {
	OverrideLevel string `json:"override_level"` // cbc_rubric_level_with_sub_levels value
}

type CalculateForClassPayload struct {
	ClassID      string `json:"class_id"`
	AcademicYear int    `json:"academic_year"`
	Term         int    `json:"term"`
}

type CalculateResponse struct {
	Count int `json:"count"`
}

type MarkSyncedPayload struct {
	KNECSyncStatus string  `json:"knec_sync_status"` // Synced | Failed
	ReferenceID    *string `json:"reference_id,omitempty"`
}

// ============================================================================
// Valid Rubric Level Sets
// ============================================================================

// ValidBaseRubricLevels is the set of valid KNEC 4-level rubric outcomes.
var ValidBaseRubricLevels = map[string]bool{
	"EE": true,
	"ME": true,
	"AE": true,
	"BE": true,
}

// ValidSubRubricLevels is the set of valid rubric levels with sub-levels.
var ValidSubRubricLevels = map[string]bool{
	"EE":  true,
	"ME":  true,
	"AE":  true,
	"BE":  true,
	"EE1": true, "EE2": true,
	"ME1": true, "ME2": true,
	"AE1": true, "AE2": true,
	"BE1": true, "BE2": true,
}

// ValidKNECSyncStatuses is the set of valid knec_sync_status values.
var ValidKNECSyncStatuses = map[string]bool{
	"Pending": true,
	"Synced":  true,
	"Failed":  true,
}

// BaseRubricLevel maps any sub-level to its base 4-level value.
func BaseRubricLevel(level string) string {
	if len(level) >= 2 {
		base := level[:2]
		if ValidBaseRubricLevels[base] {
			return base
		}
	}
	return ""
}

// ============================================================================
// Repository Interface
// ============================================================================

// Repository defines the contract for competency summary persistence.
type Repository interface {
	// CRUD
	GetByID(ctx context.Context, id, tenantID string) (*CompetencySummary, error)
	List(ctx context.Context, tenantID string, query ListSummariesQuery) ([]CompetencySummary, error)

	// Calculation: aggregates rubric results and upserts summaries
	CalculateForClass(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error)

	// Override
	SetOverrideLevel(ctx context.Context, id, tenantID string, overrideLevel *string) error

	// KNEC sync
	MarkSynced(ctx context.Context, id, tenantID, status string, syncedAt *string) error

	// Get current sync status for conflict detection
	GetSyncStatus(ctx context.Context, id, tenantID string) (string, error)
}
