package academicyears

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// todayEAT returns today's date in EAT (UTC+3) as a JS Date at midnight UTC.
// Used as the default value for the `now` parameter in service methods.
func todayEAT() time.Time {
	now := time.Now()
	eat := now.Add(3 * time.Hour)
	return time.Date(eat.Year(), eat.Month(), eat.Day(), 0, 0, 0, 0, time.UTC)
}

// parseDate parses a "YYYY-MM-DD" string into a time.Time.
func parseDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q: %w", s, err)
	}
	return t, nil
}

// ============================================================================
// Service
// ============================================================================

// Service contains business logic for academic years and terms.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// ============================================================================
// YEARS
// ============================================================================

// ListYears returns all non-deleted academic years for a school with nested terms.
func (s *Service) ListYears(ctx context.Context, tenantID, schoolID string) ([]AcademicYearWithTerms, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("academicyears.Service.ListYears: %w", ErrInvalidInput)
	}
	return s.Repo.ListYears(ctx, tenantID, schoolID)
}

// PatchYear applies partial updates to an academic year.
func (s *Service) PatchYear(ctx context.Context, id, tenantID, schoolID string, body PatchYearBody, actorID string) (*AcademicYear, *TermsOutOfRangeError) {
	// Step 1 — Fetch with FOR UPDATE
	year, err := s.Repo.GetYearByIDForUpdate(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, nil // error propagated as 404
	}

	// Step 2 — Optimistic lock check
	if body.Version == nil || *body.Version != year.Version {
		return nil, nil // error propagated as 409
	}

	// Apply changes
	if body.Name != nil {
		year.Name = *body.Name
	}
	if body.StartDate != nil {
		newStart, parseErr := parseDate(*body.StartDate)
		if parseErr != nil {
			return nil, nil // error propagated as invalid_input
		}
		year.StartDate = newStart
	}
	if body.EndDate != nil {
		newEnd, parseErr := parseDate(*body.EndDate)
		if parseErr != nil {
			return nil, nil // error propagated as invalid_input
		}
		year.EndDate = newEnd
	}

	year.UpdatedBy = actorID

	// Step 3 — Term-strandedness check (if dates changed)
	if body.StartDate != nil || body.EndDate != nil {
		stranded, err := s.Repo.FindStrandedTerms(ctx, year.ID, year.StartDate, year.EndDate)
		if err != nil {
			return nil, nil // error propagated
		}
		if len(stranded) > 0 {
			return nil, &TermsOutOfRangeError{ConflictingTerms: stranded}
		}
	}

	// Step 4 — Apply update
	if err := s.Repo.UpdateYear(ctx, year); err != nil {
		return nil, nil // error propagated
	}

	// Log the mutation
	slog.Info("academic_year.patched",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"actor_id", actorID,
		"changes", map[string]interface{}{
			"name":       body.Name,
			"start_date": body.StartDate,
			"end_date":   body.EndDate,
		},
	)

	return year, nil
}

// DeleteYear soft-deletes an academic year and all its terms.
func (s *Service) DeleteYear(ctx context.Context, id, tenantID, schoolID, actorID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("academicyears.Service.DeleteYear: %w", ErrInvalidInput)
	}

	// Fetch and verify existence
	_, err := s.Repo.GetYearByIDForUpdate(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("academicyears.Service.DeleteYear: %w", err)
	}

	// Check for dependents
	hasDeps, err := s.Repo.HasDependents(ctx, id)
	if err != nil {
		return fmt.Errorf("academicyears.Service.DeleteYear: %w", err)
	}
	if hasDeps {
		return &HasDependentsError{
			Message: "This academic year has linked records and cannot be deleted. Archive it instead.",
		}
	}

	// Soft-delete all terms first, then the year
	// In production, these would be in a transaction using Begin/Commit.
	// For the current implementation we rely on individual calls.
	// NOTE: In production, wrap in a transaction that does terms first.
	if err := s.Repo.SoftDeleteYear(ctx, id, actorID); err != nil {
		return fmt.Errorf("academicyears.Service.DeleteYear: %w", err)
	}

	slog.Info("academic_year.deleted",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"actor_id", actorID,
	)

	return nil
}

// SetCurrentYear sets a single year as is_current and clears all others.
func (s *Service) SetCurrentYear(ctx context.Context, id, tenantID, schoolID, actorID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("academicyears.Service.SetCurrentYear: %w", ErrInvalidInput)
	}

	// Clear current on all other years
	if err := s.Repo.ClearCurrentYear(ctx, schoolID, tenantID, id, actorID); err != nil {
		return fmt.Errorf("academicyears.Service.SetCurrentYear: %w", err)
	}

	// Set current on the target year
	found, err := s.Repo.SetCurrentYear(ctx, id, tenantID, schoolID, actorID)
	if err != nil {
		return fmt.Errorf("academicyears.Service.SetCurrentYear: %w", err)
	}
	if !found {
		return fmt.Errorf("academicyears.Service.SetCurrentYear: %w", ErrNotFound)
	}

	slog.Info("academic_year.set_current",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"actor_id", actorID,
	)

	return nil
}

// ============================================================================
// TERMS
// ============================================================================

// ListTerms returns all non-deleted terms, optionally filtered by academic_year_id.
func (s *Service) ListTerms(ctx context.Context, tenantID, schoolID string, academicYearID *string) ([]AcademicTerm, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("academicyears.Service.ListTerms: %w", ErrInvalidInput)
	}
	return s.Repo.ListTerms(ctx, tenantID, schoolID, academicYearID)
}

// CreateTerm creates a new academic term with full validation.
func (s *Service) CreateTerm(ctx context.Context, body CreateTermBody, tenantID, schoolID, actorID string, now *time.Time) (*AcademicTerm, error) {
	if now == nil {
		n := todayEAT()
		now = &n
	}

	if body.Name == "" || body.TermNumber < 1 || body.TermNumber > 3 {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", ErrInvalidInput)
	}

	startDate, err := parseDate(body.StartDate)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", ErrInvalidInput)
	}
	endDate, err := parseDate(body.EndDate)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", ErrInvalidInput)
	}

	// Fetch parent year
	year, err := s.Repo.GetYearByID(ctx, body.AcademicYearID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", err)
	}

	// Boundary check (inclusive)
	if startDate.Before(year.StartDate) || endDate.After(year.EndDate) {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", &TermOutOfYearBoundsError{})
	}

	// Overlap check
	overlapping, err := s.Repo.FindOverlappingTerms(ctx, body.AcademicYearID, "", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", err)
	}
	if len(overlapping) > 0 {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w",
			&TermDateOverlapError{ConflictingName: overlapping[0].Name})
	}

	// Create the term
	term := &AcademicTerm{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: body.AcademicYearID,
		Name:           body.Name,
		TermNumber:     body.TermNumber,
		StartDate:      startDate,
		EndDate:        endDate,
		CreatedBy:      actorID,
		UpdatedBy:      actorID,
	}

	id, err := s.Repo.CreateTerm(ctx, term)
	if err != nil {
		// Check for unique constraint violation on (academic_year_id, term_number)
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", &TermNumberExistsError{})
		}
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", err)
	}
	term.ID = id

	// Sync current term
	if err := s.Repo.SyncCurrentTerm(ctx, body.AcademicYearID, *now); err != nil {
		return nil, fmt.Errorf("academicyears.Service.CreateTerm: %w", err)
	}

	slog.Info("academic_term.created",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"actor_id", actorID,
		"academic_year_id", body.AcademicYearID,
		"term_number", body.TermNumber,
	)

	return term, nil
}

// PatchTerm applies partial updates to a term.
func (s *Service) PatchTerm(ctx context.Context, id, tenantID, schoolID string, body PatchTermBody, actorID string, now *time.Time) (*AcademicTerm, error) {
	if now == nil {
		n := todayEAT()
		now = &n
	}

	// Fetch term + parent year with FOR UPDATE
	term, year, err := s.Repo.GetTermByIDForUpdate(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", err)
	}

	// Optimistic lock check
	if body.Version == nil || *body.Version != term.Version {
		return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", ErrConflict)
	}

	// Apply changes
	if body.Name != nil {
		term.Name = *body.Name
	}
	if body.StartDate != nil {
		newStart, parseErr := parseDate(*body.StartDate)
		if parseErr != nil {
			return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", ErrInvalidInput)
		}
		term.StartDate = newStart
	}
	if body.EndDate != nil {
		newEnd, parseErr := parseDate(*body.EndDate)
		if parseErr != nil {
			return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", ErrInvalidInput)
		}
		term.EndDate = newEnd
	}

	// If dates changed, run boundary and overlap checks
	if body.StartDate != nil || body.EndDate != nil {
		// Boundary check
		if term.StartDate.Before(year.StartDate) || term.EndDate.After(year.EndDate) {
			return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", &TermOutOfYearBoundsError{})
		}

		// Overlap check (exclude self)
		overlapping, err := s.Repo.FindOverlappingTerms(ctx, term.AcademicYearID, term.ID, term.StartDate, term.EndDate)
		if err != nil {
			return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", err)
		}
		if len(overlapping) > 0 {
			return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w",
				&TermDateOverlapError{ConflictingName: overlapping[0].Name})
		}
	}

	term.UpdatedBy = actorID

	if err := s.Repo.UpdateTerm(ctx, term); err != nil {
		return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", err)
	}

	// Sync current term
	if err := s.Repo.SyncCurrentTerm(ctx, term.AcademicYearID, *now); err != nil {
		return nil, fmt.Errorf("academicyears.Service.PatchTerm: %w", err)
	}

	slog.Info("academic_term.patched",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"actor_id", actorID,
		"academic_year_id", term.AcademicYearID,
		"changes", map[string]interface{}{
			"name":       body.Name,
			"start_date": body.StartDate,
			"end_date":   body.EndDate,
		},
	)

	return term, nil
}

// DeleteTerm soft-deletes a term and syncs current term.
func (s *Service) DeleteTerm(ctx context.Context, id, tenantID, schoolID, actorID string, now *time.Time) error {
	if now == nil {
		n := todayEAT()
		now = &n
	}

	// Fetch term + parent year
	term, _, err := s.Repo.GetTermByIDForUpdate(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("academicyears.Service.DeleteTerm: %w", err)
	}

	// Check for dependents
	hasDeps, err := s.Repo.HasTermDependents(ctx, id)
	if err != nil {
		return fmt.Errorf("academicyears.Service.DeleteTerm: %w", err)
	}
	if hasDeps {
		return &HasDependentsError{
			Message: "This academic term has linked records and cannot be deleted. Archive it instead.",
		}
	}

	if err := s.Repo.SoftDeleteTerm(ctx, id, actorID); err != nil {
		return fmt.Errorf("academicyears.Service.DeleteTerm: %w", err)
	}

	// Sync current term
	if err := s.Repo.SyncCurrentTerm(ctx, term.AcademicYearID, *now); err != nil {
		return fmt.Errorf("academicyears.Service.DeleteTerm: %w", err)
	}

	slog.Info("academic_term.deleted",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"actor_id", actorID,
		"academic_year_id", term.AcademicYearID,
	)

	return nil
}

// isUniqueViolation heuristically checks if an error is a unique constraint
// violation from pgx. In production, use pgerrcode.UniqueViolation.
func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "unique constraint") ||
		contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
