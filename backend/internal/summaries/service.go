package summaries

import (
	"context"
	"fmt"
	"log/slog"
)

// ============================================================================
// Service
// ============================================================================

// Service contains business logic for competency summaries.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// validateRubricLevelWithSubLevels checks if a string is a valid
// cbc_rubric_level_with_sub_levels value.
func validateRubricLevelWithSubLevels(level string) bool {
	return ValidSubRubricLevels[level]
}

// validateKNECSyncStatus checks if a string is a valid knec_sync_status value.
func validateKNECSyncStatus(status string) bool {
	return ValidKNECSyncStatuses[status]
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID returns a single summary by ID.
func (s *Service) GetByID(ctx context.Context, id, tenantID string) (*CompetencySummary, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("summaries.Service.GetByID: %w", ErrInvalidInput)
	}
	return s.Repo.GetByID(ctx, id, tenantID)
}

// ============================================================================
// List
// ============================================================================

// List returns summaries matching the given filters.
func (s *Service) List(ctx context.Context, tenantID string, query ListSummariesQuery) ([]CompetencySummary, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("summaries.Service.List: %w", ErrInvalidInput)
	}
	return s.Repo.List(ctx, tenantID, query)
}

// ============================================================================
// CalculateForClass
// ============================================================================

// CalculateForClass runs the modal rubric level aggregation for all students
// in a given class + term and upserts the summaries. Returns the count of
// summaries created/updated.
func (s *Service) CalculateForClass(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
	if tenantID == "" {
		return 0, fmt.Errorf("summaries.Service.CalculateForClass: %w", ErrInvalidInput)
	}
	if payload.ClassID == "" {
		return 0, fmt.Errorf("summaries.Service.CalculateForClass: %w", ErrInvalidInput)
	}
	if payload.AcademicYear < 2017 {
		return 0, fmt.Errorf("summaries.Service.CalculateForClass: academic_year must be >= 2017: %w", ErrInvalidInput)
	}
	if payload.Term < 1 || payload.Term > 3 {
		return 0, fmt.Errorf("summaries.Service.CalculateForClass: term must be 1-3: %w", ErrInvalidInput)
	}

	count, err := s.Repo.CalculateForClass(ctx, tenantID, payload)
	if err != nil {
		return 0, fmt.Errorf("summaries.Service.CalculateForClass: %w", err)
	}

	slog.Info("summaries.calculated",
		"tenant_id", tenantID,
		"class_id", payload.ClassID,
		"academic_year", payload.AcademicYear,
		"term", payload.Term,
		"summary_count", count,
	)

	return count, nil
}

// ============================================================================
// SetOverrideLevel
// ============================================================================

// SetOverrideLevel sets or clears the override_level for a summary.
// If overrideLevel is empty string, the override is cleared.
// Validates that the override level is a valid cbc_rubric_level_with_sub_levels value.
func (s *Service) SetOverrideLevel(ctx context.Context, id, tenantID string, payload OverrideLevelPayload) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("summaries.Service.SetOverrideLevel: %w", ErrInvalidInput)
	}

	// Validate the override level
	if payload.OverrideLevel != "" {
		if !validateRubricLevelWithSubLevels(payload.OverrideLevel) {
			return fmt.Errorf("summaries.Service.SetOverrideLevel: invalid override_level %q: %w",
				payload.OverrideLevel, ErrInvalidRubricLevel)
		}
	}

	// Convert empty string to nil pointer to clear override
	var overridePtr *string
	if payload.OverrideLevel != "" {
		overridePtr = &payload.OverrideLevel
	}

	// Fetch current summary to check if it's already synced (overrides not allowed on synced records)
	summary, err := s.Repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("summaries.Service.SetOverrideLevel: %w", err)
	}

	if summary.KNECSyncStatus == "Synced" {
		return fmt.Errorf("summaries.Service.SetOverrideLevel: cannot override a synced summary: %w", ErrConflict)
	}

	if err := s.Repo.SetOverrideLevel(ctx, id, tenantID, overridePtr); err != nil {
		return fmt.Errorf("summaries.Service.SetOverrideLevel: %w", err)
	}

	slog.Info("summaries.override.set",
		"tenant_id", tenantID,
		"summary_id", id,
		"override_level", payload.OverrideLevel,
	)

	return nil
}

// ============================================================================
// MarkSynced
// ============================================================================

// MarkSynced updates the KNEC sync status for a summary.
// Enforces: Pending → Synced, Pending → Failed, Failed → Synced, but not Synced → any.
func (s *Service) MarkSynced(ctx context.Context, id, tenantID string, payload MarkSyncedPayload) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("summaries.Service.MarkSynced: %w", ErrInvalidInput)
	}

	// Validate sync status
	if !validateKNECSyncStatus(payload.KNECSyncStatus) {
		return fmt.Errorf("summaries.Service.MarkSynced: invalid knec_sync_status %q: %w",
			payload.KNECSyncStatus, ErrInvalidInput)
	}

	// Only Synced and Failed are valid target states for this operation
	if payload.KNECSyncStatus != "Synced" && payload.KNECSyncStatus != "Failed" {
		return fmt.Errorf("summaries.Service.MarkSynced: target status must be Synced or Failed: %w", ErrInvalidInput)
	}

	// Check current status to prevent re-syncing
	currentStatus, err := s.Repo.GetSyncStatus(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("summaries.Service.MarkSynced: %w", err)
	}

	if currentStatus == "Synced" && payload.KNECSyncStatus == "Synced" {
		return fmt.Errorf("summaries.Service.MarkSynced: summary is already synced: %w", ErrAlreadySynced)
	}

	var syncedAt *string
	if payload.KNECSyncStatus == "Synced" {
		ts := "__SET_BY_DB__" // Placeholder; the repository uses NOW()
		syncedAt = &ts
	}

	if err := s.Repo.MarkSynced(ctx, id, tenantID, payload.KNECSyncStatus, syncedAt); err != nil {
		return fmt.Errorf("summaries.Service.MarkSynced: %w", err)
	}

	slog.Info("summaries.sync.updated",
		"tenant_id", tenantID,
		"summary_id", id,
		"status", payload.KNECSyncStatus,
		"reference_id", payload.ReferenceID,
	)

	return nil
}
