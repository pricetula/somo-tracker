package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	storegen "somotracker/internal/schools/store/gen"
	"somotracker/internal/platform/apperr"
)

// Store implements the schools domain data layer.
// SQL lives in queries/*.sql, generated to gen/ by sqlc (BE-009).
type Store struct {
	q *storegen.Queries
}

// New creates store.
func New(q *storegen.Queries) *Store {
	return &Store{q: q}
}

// mapPGError converts Postgres errors to apperr errors.
func mapPGError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.New(apperr.NOT_FOUND, "school not found")
	}
	// Check for unique violation (23505)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CONFLICT, "school code already exists in tenant")
		}
	}
	return apperr.New(apperr.INTERNAL_ERROR, "database error")
}

// GetSchool returns a single school by ID, tenant-scoped.
func (s *Store) GetSchool(ctx context.Context, id uuid.UUID) (*storegen.School, error) {
	row, err := s.q.GetSchool(ctx, id)
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// ListSchools returns schools with keyset pagination, tenant-scoped.
// cursorName and cursorID encode the last item from the previous page.
func (s *Store) ListSchools(ctx context.Context, limit int32, cursorName string, cursorID uuid.UUID) ([]storegen.School, error) {
	rows, err := s.q.ListSchools(ctx, storegen.ListSchoolsParams{
		Limit:   limit,
		Column2: cursorName,
		Column3: cursorID,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return rows, nil
}

// CreateSchool inserts a new school.
func (s *Store) CreateSchool(ctx context.Context, tenantID uuid.UUID, name string, code string, countryID uuid.UUID, educationSystemID uuid.UUID) (*storegen.School, error) {
	row, err := s.q.CreateSchool(ctx, storegen.CreateSchoolParams{
		TenantID:          tenantID,
		Name:              name,
		Code:              code,
		CountryID:         countryID,
		EducationSystemID: educationSystemID,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// UpdateSchool updates an existing school (full body).
func (s *Store) UpdateSchool(ctx context.Context, id uuid.UUID, name string, code string, countryID uuid.UUID, educationSystemID uuid.UUID) (*storegen.School, error) {
	row, err := s.q.UpdateSchool(ctx, storegen.UpdateSchoolParams{
		ID:                id,
		Name:              name,
		Code:              code,
		CountryID:         countryID,
		EducationSystemID: educationSystemID,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// DeactivateSchools soft-deletes schools by IDs.
func (s *Store) DeactivateSchools(ctx context.Context, ids []uuid.UUID) error {
	err := s.q.DeactivateSchools(ctx, ids)
	return mapPGError(err)
}
