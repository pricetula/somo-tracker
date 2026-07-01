package portfolio

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles portfolio entry database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// isFKViolation checks if an error is a PostgreSQL foreign key violation (23503).
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

// ============================================================================
// Cross-Domain Resolver Implementations
// ============================================================================

// StudentExists checks whether a student exists in the given tenant.
// Implements portfolio.StudentResolver.
func (r *PgRepository) StudentExists(ctx context.Context, tenantID, studentID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_students
			WHERE id = $1 AND tenant_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, studentID, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("portfolio.Repository.StudentExists: %w", err)
	}
	return exists, nil
}

// SubStrandExists checks whether a sub-strand exists.
// Implements portfolio.SubStrandResolver.
func (r *PgRepository) SubStrandExists(ctx context.Context, subStrandID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_sub_strands
			WHERE id = $1
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, subStrandID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("portfolio.Repository.SubStrandExists: %w", err)
	}
	return exists, nil
}

// ============================================================================
// CRUD
// ============================================================================

// CreateEntry inserts a new portfolio entry and returns its ID.
func (r *PgRepository) CreateEntry(ctx context.Context, e *PortfolioEntry) (string, error) {
	const query = `
		INSERT INTO learner_portfolios
			(tenant_id, student_id, sub_strand_id, evidence_type, storage_pointer, linked_result_id, date_collected)
		VALUES ($1, $2, $3, $4::portfolio_evidence_type, $5, $6, $7::date)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		e.TenantID, e.StudentID, e.SubStrandID,
		e.EvidenceType, e.StoragePointer,
		e.LinkedResultID, e.DateCollected,
	).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return "", fmt.Errorf("portfolio.Repository.CreateEntry: %w", ErrNotFound)
		}
		return "", fmt.Errorf("portfolio.Repository.CreateEntry: %w", err)
	}
	return id, nil
}

// GetEntryByID retrieves a single portfolio entry by primary key and tenant.
func (r *PgRepository) GetEntryByID(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
	const query = `
		SELECT id, tenant_id, student_id, sub_strand_id,
		       evidence_type::text, storage_pointer, linked_result_id,
		       date_collected::text, created_at::text
		FROM learner_portfolios
		WHERE id = $1 AND tenant_id = $2
	`
	var e PortfolioEntry
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&e.ID, &e.TenantID, &e.StudentID, &e.SubStrandID,
		&e.EvidenceType, &e.StoragePointer, &e.LinkedResultID,
		&e.DateCollected, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("portfolio.Repository.GetEntryByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("portfolio.Repository.GetEntryByID: %w", err)
	}
	return &e, nil
}

// ListEntries returns portfolio entries filtered by the given query parameters.
func (r *PgRepository) ListEntries(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error) {
	baseQuery := `
		SELECT id, tenant_id, student_id, sub_strand_id,
		       evidence_type::text, storage_pointer, linked_result_id,
		       date_collected::text, created_at::text
		FROM learner_portfolios
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if query.StudentID != "" {
		baseQuery += fmt.Sprintf(" AND student_id = $%d", argIdx)
		args = append(args, query.StudentID)
		argIdx++
	}
	if query.SubStrandID != "" {
		baseQuery += fmt.Sprintf(" AND sub_strand_id = $%d", argIdx)
		args = append(args, query.SubStrandID)
	}

	baseQuery += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("portfolio.Repository.ListEntries: %w", err)
	}
	defer rows.Close()

	var entries []PortfolioEntry
	for rows.Next() {
		var e PortfolioEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.StudentID, &e.SubStrandID,
			&e.EvidenceType, &e.StoragePointer, &e.LinkedResultID,
			&e.DateCollected, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("portfolio.Repository.ListEntries: scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portfolio.Repository.ListEntries: rows: %w", err)
	}
	if entries == nil {
		entries = []PortfolioEntry{}
	}
	return entries, nil
}

// UpdateEntry applies partial updates to a portfolio entry.
func (r *PgRepository) UpdateEntry(ctx context.Context, e *PortfolioEntry) error {
	const query = `
		UPDATE learner_portfolios
		SET storage_pointer = $1,
		    date_collected   = $2::date,
		    linked_result_id = $3
		WHERE id = $4 AND tenant_id = $5
	`
	tag, err := r.pool.Exec(ctx, query,
		e.StoragePointer, e.DateCollected, e.LinkedResultID,
		e.ID, e.TenantID,
	)
	if err != nil {
		if isFKViolation(err) {
			return fmt.Errorf("portfolio.Repository.UpdateEntry: %w", ErrNotFound)
		}
		return fmt.Errorf("portfolio.Repository.UpdateEntry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("portfolio.Repository.UpdateEntry: %w", ErrNotFound)
	}
	return nil
}

// DeleteEntry removes a portfolio entry by ID.
func (r *PgRepository) DeleteEntry(ctx context.Context, id, tenantID string) error {
	const query = `
		DELETE FROM learner_portfolios
		WHERE id = $1 AND tenant_id = $2
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("portfolio.Repository.DeleteEntry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("portfolio.Repository.DeleteEntry: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// Duplicate Advisory Check
// ============================================================================

// EntryExistsForStudentSubStrandEvidence checks whether a portfolio entry
// already exists for the given (student, sub_strand, evidence_type) combination.
// This is advisory only — no DB unique constraint blocks duplicates.
func (r *PgRepository) EntryExistsForStudentSubStrandEvidence(ctx context.Context, tenantID, studentID, subStrandID, evidenceType string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM learner_portfolios
			WHERE tenant_id = $1
			  AND student_id = $2
			  AND sub_strand_id = $3
			  AND evidence_type = $4::portfolio_evidence_type
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, tenantID, studentID, subStrandID, evidenceType).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("portfolio.Repository.EntryExistsForStudentSubStrandEvidence: %w", err)
	}
	return exists, nil
}
