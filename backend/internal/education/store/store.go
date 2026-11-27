package store

import (
	"context"

	"github.com/google/uuid"
	storegen "somotracker/internal/auth/store/gen"
)

// Store wraps sqlc-generated queries for the education domain.
type Store struct {
	q *storegen.Queries
}

// New creates an education store.
func New(q *storegen.Queries) *Store {
	return &Store{q: q}
}

// ListCountries returns all countries ordered by name.
func (s *Store) ListCountries(ctx context.Context) ([]storegen.Country, error) {
	return s.q.ListCountries(ctx)
}

// GetCountry returns a single country by ID.
func (s *Store) GetCountry(ctx context.Context, id uuid.UUID) (storegen.Country, error) {
	return s.q.GetCountry(ctx, id)
}

// ListEducationSystems returns all education systems with country info.
func (s *Store) ListEducationSystems(ctx context.Context) ([]storegen.ListEducationSystemsRow, error) {
	return s.q.ListEducationSystems(ctx)
}

// GetEducationSystem returns a single education system with country info.
func (s *Store) GetEducationSystem(ctx context.Context, id uuid.UUID) (storegen.GetEducationSystemRow, error) {
	return s.q.GetEducationSystem(ctx, id)
}

// ListEducationSystemLevels returns all levels for an education system.
func (s *Store) ListEducationSystemLevels(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemLevelsRow, error) {
	return s.q.ListEducationSystemLevels(ctx, educationSystemID)
}

// ListEducationSystemGradingBands returns all grading bands for an education system.
func (s *Store) ListEducationSystemGradingBands(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemGradingBandsRow, error) {
	return s.q.ListEducationSystemGradingBands(ctx, educationSystemID)
}

// ListEducationSystemSubjects returns all subjects for an education system.
func (s *Store) ListEducationSystemSubjects(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemSubjectsRow, error) {
	return s.q.ListEducationSystemSubjects(ctx, educationSystemID)
}

