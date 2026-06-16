package academiccalendar

import (
	"context"
	"fmt"
)

// Service contains business logic for the academic calendar.
type Service struct {
	repo *Repository
}

// NewService creates a new Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ResolveSchoolID returns the primary active school ID for a tenant.
// Falls back to the first active school found.
func (s *Service) ResolveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	schoolID, err := s.repo.GetPrimarySchoolID(ctx, tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("resolve school: %w", err)
	}
	return schoolID, nil
}

// GetCurrentCalendar returns the current academic calendar for a school.
func (s *Service) GetCurrentCalendar(ctx context.Context, schoolID, tenantID string) (*AcademicYear, error) {
	return s.repo.GetCurrentCalendar(ctx, schoolID, tenantID)
}

// SaveCurrentCalendar creates or replaces the current academic calendar for a school.
// This is an atomic transaction: it unsets the old current year, upserts the new
// year record, and replaces all its terms.
func (s *Service) SaveCurrentCalendar(
	ctx context.Context,
	schoolID, tenantID string,
	payload SavePayload,
) (*AcademicYear, error) {
	if len(payload.Periods) == 0 {
		return nil, fmt.Errorf("at least one period is required")
	}

	// Derive the overall year date range from the periods
	firstStart := payload.Periods[0].StartDate
	lastEnd := payload.Periods[len(payload.Periods)-1].EndDate

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	// 1. Unset any existing current year for this school
	if err := s.repo.UnsetCurrentYears(ctx, tx, schoolID, tenantID); err != nil {
		return nil, fmt.Errorf("unset current years: %w", err)
	}

	// 2. Upsert the year record
	yearID, err := s.repo.UpsertYear(ctx, tx, schoolID, tenantID, payload.Year, firstStart, lastEnd)
	if err != nil {
		return nil, fmt.Errorf("upsert year: %w", err)
	}

	// 3. Replace all terms
	if err := s.repo.ReplaceTerms(ctx, tx, yearID, tenantID, payload.Periods); err != nil {
		return nil, fmt.Errorf("replace terms: %w", err)
	}

	// 4. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// 5. Fetch and return the full saved calendar
	return s.repo.GetCurrentCalendar(ctx, schoolID, tenantID)
}
