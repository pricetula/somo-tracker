package academicyears

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles academic year and term database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ============================================================================
// Transaction helpers
// ============================================================================

// Begin starts a PostgreSQL transaction and stores it in the context.
func (r *PgRepository) Begin(ctx context.Context) (Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.Begin: %w", err)
	}
	return &pgTx{tx: tx}, nil
}

type pgTx struct {
	tx pgx.Tx
}

func (t *pgTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// ============================================================================
// YEARS
// ============================================================================

// ListYears returns all non-deleted academic years for a school, with nested
// terms ordered by term_number.
func (r *PgRepository) ListYears(ctx context.Context, tenantID, schoolID string) ([]AcademicYearWithTerms, error) {
	const query = `
		SELECT
			ay.id, ay.tenant_id, ay.school_id, ay.name,
			ay.start_date, ay.end_date, ay.is_current,
			ay.version, ay.created_by, ay.updated_by,
			ay.created_at, ay.updated_at,
			COALESCE(
				json_agg(
					json_build_object(
						'id', at.id,
						'name', at.name,
						'term_number', at.term_number,
						'start_date', at.start_date,
						'end_date', at.end_date,
						'is_current', at.is_current,
						'version', at.version,
						'created_at', at.created_at,
						'updated_at', at.updated_at
					) ORDER BY at.term_number ASC
				) FILTER (WHERE at.id IS NOT NULL AND at.deleted_at IS NULL),
				'[]'
			) AS terms
		FROM academic_years ay
		LEFT JOIN academic_terms at ON at.academic_year_id = ay.id
		WHERE ay.tenant_id = $1
		  AND ay.school_id = $2
		  AND ay.deleted_at IS NULL
		GROUP BY ay.id
		ORDER BY ay.start_date DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ListYears: %w", err)
	}
	defer rows.Close()

	var years []AcademicYearWithTerms
	for rows.Next() {
		var y AcademicYearWithTerms
		if err := rows.Scan(
			&y.ID, &y.TenantID, &y.SchoolID, &y.Name,
			&y.StartDate, &y.EndDate, &y.IsCurrent,
			&y.Version, &y.CreatedBy, &y.UpdatedBy,
			&y.CreatedAt, &y.UpdatedAt, &y.Terms,
		); err != nil {
			return nil, fmt.Errorf("academicyears.Repository.ListYears: scan: %w", err)
		}
		years = append(years, y)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ListYears: rows: %w", err)
	}
	if years == nil {
		years = []AcademicYearWithTerms{}
	}
	return years, nil
}

// GetYearByID retrieves a single non-deleted year by primary key.
func (r *PgRepository) GetYearByID(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
	const query = `
		SELECT id, tenant_id, school_id, name,
		       start_date, end_date, is_current,
		       version, created_by, updated_by,
		       created_at, updated_at
		FROM academic_years
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3 AND deleted_at IS NULL
	`
	var y AcademicYear
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&y.ID, &y.TenantID, &y.SchoolID, &y.Name,
		&y.StartDate, &y.EndDate, &y.IsCurrent,
		&y.Version, &y.CreatedBy, &y.UpdatedBy,
		&y.CreatedAt, &y.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("academicyears.Repository.GetYearByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("academicyears.Repository.GetYearByID: %w", err)
	}
	return &y, nil
}

// GetYearByIDForUpdate retrieves a year with FOR UPDATE row locking.
func (r *PgRepository) GetYearByIDForUpdate(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
	const query = `
		SELECT id, tenant_id, school_id, name,
		       start_date, end_date, is_current,
		       version, created_by, updated_by,
		       created_at, updated_at
		FROM academic_years
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3 AND deleted_at IS NULL
		FOR UPDATE
	`
	var y AcademicYear
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&y.ID, &y.TenantID, &y.SchoolID, &y.Name,
		&y.StartDate, &y.EndDate, &y.IsCurrent,
		&y.Version, &y.CreatedBy, &y.UpdatedBy,
		&y.CreatedAt, &y.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("academicyears.Repository.GetYearByIDForUpdate: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("academicyears.Repository.GetYearByIDForUpdate: %w", err)
	}
	return &y, nil
}

// CreateYear inserts a new academic year and returns its ID.
func (r *PgRepository) CreateYear(ctx context.Context, year *AcademicYear) (string, error) {
	const query = `
		INSERT INTO academic_years (tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		year.TenantID, year.SchoolID, year.Name,
		year.StartDate, year.EndDate,
		year.CreatedBy, year.UpdatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("academicyears.Repository.CreateYear: %w", err)
	}
	return id, nil
}

// UpdateYear applies changes to a year, incrementing version.
func (r *PgRepository) UpdateYear(ctx context.Context, year *AcademicYear) error {
	const query = `
		UPDATE academic_years
		SET name = $1, start_date = $2, end_date = $3,
		    version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE id = $5 AND version = $6 AND deleted_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query,
		year.Name, year.StartDate, year.EndDate,
		year.UpdatedBy, year.ID, year.Version,
	)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.UpdateYear: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Could be not found or version mismatch — check which
		existing, checkErr := r.GetYearByID(ctx, year.ID, year.TenantID, year.SchoolID)
		if checkErr != nil {
			return fmt.Errorf("academicyears.Repository.UpdateYear: %w", ErrNotFound)
		}
		if existing.Version != year.Version {
			return fmt.Errorf("academicyears.Repository.UpdateYear: %w", ErrConflict)
		}
		return fmt.Errorf("academicyears.Repository.UpdateYear: %w", ErrNotFound)
	}
	return nil
}

// SoftDeleteYear sets deleted_at on a year.
func (r *PgRepository) SoftDeleteYear(ctx context.Context, id, actorID string) error {
	const query = `
		UPDATE academic_years
		SET deleted_at = NOW(), updated_by = $2, version = version + 1
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, id, actorID)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.SoftDeleteYear: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("academicyears.Repository.SoftDeleteYear: %w", ErrNotFound)
	}
	return nil
}

// ClearCurrentYear sets is_current = FALSE for all years in a school except the
// specified excludeID. Used inside setCurrentYear transaction.
func (r *PgRepository) ClearCurrentYear(ctx context.Context, schoolID, tenantID, excludeID, actorID string) error {
	const query = `
		UPDATE academic_years
		SET is_current = FALSE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE school_id = $1 AND tenant_id = $2 AND is_current = TRUE
		  AND deleted_at IS NULL AND id != $3
	`
	_, err := r.pool.Exec(ctx, query, schoolID, tenantID, excludeID, actorID)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.ClearCurrentYear: %w", err)
	}
	return nil
}

// SetCurrentYear sets is_current = TRUE on a single year. Returns true if a row
// was updated, false otherwise (which translates to 404).
func (r *PgRepository) SetCurrentYear(ctx context.Context, id, tenantID, schoolID, actorID string) (bool, error) {
	const query = `
		UPDATE academic_years
		SET is_current = TRUE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3 AND deleted_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID, schoolID, actorID)
	if err != nil {
		return false, fmt.Errorf("academicyears.Repository.SetCurrentYear: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ============================================================================
// TERMS
// ============================================================================

// ListTerms returns all non-deleted terms, optionally filtered by academic_year_id.
func (r *PgRepository) ListTerms(ctx context.Context, tenantID, schoolID string, academicYearID *string) ([]AcademicTerm, error) {
	const query = `
		SELECT at.id, at.tenant_id, at.school_id, at.academic_year_id,
		       at.name, at.term_number, at.start_date, at.end_date,
		       at.is_current, at.is_final, at.version,
		       at.created_by, at.updated_by, at.created_at, at.updated_at
		FROM academic_terms at
		JOIN academic_years ay ON ay.id = at.academic_year_id
		WHERE ay.tenant_id = $1
		  AND ay.school_id = $2
		  AND at.deleted_at IS NULL
		  AND ay.deleted_at IS NULL
		  AND ($3::uuid IS NULL OR at.academic_year_id = $3)
		ORDER BY ay.start_date DESC, at.term_number ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ListTerms: %w", err)
	}
	defer rows.Close()

	var terms []AcademicTerm
	for rows.Next() {
		var t AcademicTerm
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.SchoolID, &t.AcademicYearID,
			&t.Name, &t.TermNumber, &t.StartDate, &t.EndDate,
			&t.IsCurrent, &t.IsFinal, &t.Version,
			&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("academicyears.Repository.ListTerms: scan: %w", err)
		}
		terms = append(terms, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ListTerms: rows: %w", err)
	}
	if terms == nil {
		terms = []AcademicTerm{}
	}
	return terms, nil
}

// GetTermByIDForUpdate fetches a term and its parent year with row locking.
func (r *PgRepository) GetTermByIDForUpdate(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
	const query = `
		SELECT at.id, at.tenant_id, at.school_id, at.academic_year_id,
		       at.name, at.term_number, at.start_date, at.end_date,
		       at.is_current, at.is_final, at.version,
		       at.created_by, at.updated_by, at.created_at, at.updated_at,
		       ay.id, ay.tenant_id, ay.school_id, ay.name,
		       ay.start_date, ay.end_date, ay.is_current,
		       ay.version, ay.created_by, ay.updated_by,
		       ay.created_at, ay.updated_at
		FROM academic_terms at
		JOIN academic_years ay ON ay.id = at.academic_year_id
		WHERE at.id = $1
		  AND ay.tenant_id = $2
		  AND ay.school_id = $3
		  AND at.deleted_at IS NULL
		  AND ay.deleted_at IS NULL
		FOR UPDATE OF at
	`

	var t AcademicTerm
	var y AcademicYear
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&t.ID, &t.TenantID, &t.SchoolID, &t.AcademicYearID,
		&t.Name, &t.TermNumber, &t.StartDate, &t.EndDate,
		&t.IsCurrent, &t.IsFinal, &t.Version,
		&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
		&y.ID, &y.TenantID, &y.SchoolID, &y.Name,
		&y.StartDate, &y.EndDate, &y.IsCurrent,
		&y.Version, &y.CreatedBy, &y.UpdatedBy,
		&y.CreatedAt, &y.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("academicyears.Repository.GetTermByIDForUpdate: %w", ErrNotFound)
		}
		return nil, nil, fmt.Errorf("academicyears.Repository.GetTermByIDForUpdate: %w", err)
	}
	return &t, &y, nil
}

// CreateTerm inserts a new academic term and returns its ID.
func (r *PgRepository) CreateTerm(ctx context.Context, term *AcademicTerm) (string, error) {
	const query = `
		INSERT INTO academic_terms (tenant_id, school_id, academic_year_id, name,
		                            term_number, start_date, end_date,
		                            created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		term.TenantID, term.SchoolID, term.AcademicYearID,
		term.Name, term.TermNumber, term.StartDate, term.EndDate,
		term.CreatedBy, term.UpdatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("academicyears.Repository.CreateTerm: %w", err)
	}
	return id, nil
}

// UpdateTerm applies changes to a term, incrementing version.
func (r *PgRepository) UpdateTerm(ctx context.Context, term *AcademicTerm) error {
	const query = `
		UPDATE academic_terms
		SET name = $1, start_date = $2, end_date = $3,
		    version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE id = $5 AND version = $6 AND deleted_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query,
		term.Name, term.StartDate, term.EndDate,
		term.UpdatedBy, term.ID, term.Version,
	)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.UpdateTerm: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("academicyears.Repository.UpdateTerm: %w", ErrNotFound)
	}
	return nil
}

// SoftDeleteTerm sets deleted_at on a term.
func (r *PgRepository) SoftDeleteTerm(ctx context.Context, id, actorID string) error {
	const query = `
		UPDATE academic_terms
		SET deleted_at = NOW(), updated_by = $2, version = version + 1
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, id, actorID)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.SoftDeleteTerm: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("academicyears.Repository.SoftDeleteTerm: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// BUSINESS LOGIC CHECKS
// ============================================================================

// FindStrandedTerms returns terms that would fall outside a new date range for
// the parent year.
func (r *PgRepository) FindStrandedTerms(ctx context.Context, yearID string, newStart, newEnd time.Time) ([]ConflictingTerm, error) {
	const query = `
		SELECT id, name, start_date::text, end_date::text
		FROM academic_terms
		WHERE academic_year_id = $1
		  AND deleted_at IS NULL
		  AND (start_date < $2 OR end_date > $3)
	`
	rows, err := r.pool.Query(ctx, query, yearID, newStart, newEnd)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.FindStrandedTerms: %w", err)
	}
	defer rows.Close()

	var terms []ConflictingTerm
	for rows.Next() {
		var ct ConflictingTerm
		if err := rows.Scan(&ct.ID, &ct.Name, &ct.StartDate, &ct.EndDate); err != nil {
			return nil, fmt.Errorf("academicyears.Repository.FindStrandedTerms: scan: %w", err)
		}
		terms = append(terms, ct)
	}
	return terms, rows.Err()
}

// FindOverlappingTerms returns terms whose date ranges overlap with the given
// range, optionally excluding a specific term ID (for PATCH self-exclusion).
func (r *PgRepository) FindOverlappingTerms(ctx context.Context, yearID, excludeID string, startDate, endDate time.Time) ([]AcademicTerm, error) {
	const query = `
		SELECT id, name, term_number, start_date, end_date
		FROM academic_terms
		WHERE academic_year_id = $1
		  AND deleted_at IS NULL
		  AND start_date < $3
		  AND end_date > $2
		  AND ($4::uuid IS NULL OR id != $4)
	`

	rows, err := r.pool.Query(ctx, query, yearID, startDate, endDate, nullableUUID(excludeID))
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.FindOverlappingTerms: %w", err)
	}
	defer rows.Close()

	var terms []AcademicTerm
	for rows.Next() {
		var t AcademicTerm
		if err := rows.Scan(&t.ID, &t.Name, &t.TermNumber, &t.StartDate, &t.EndDate); err != nil {
			return nil, fmt.Errorf("academicyears.Repository.FindOverlappingTerms: scan: %w", err)
		}
		terms = append(terms, t)
	}
	return terms, rows.Err()
}

// HasDependents checks if any FK-referencing tables have rows linked to this year.
func (r *PgRepository) HasDependents(ctx context.Context, academicYearID string) (bool, error) {
	// Check multiple referencing tables
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_classes WHERE academic_year_id = $1
			UNION ALL
			SELECT 1 FROM cbc_timetable_slots WHERE academic_year_id = $1
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, academicYearID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("academicyears.Repository.HasDependents: %w", err)
	}
	return exists, nil
}

// HasTermDependents checks if any FK-referencing tables have rows linked to this term.
func (r *PgRepository) HasTermDependents(ctx context.Context, termID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_student_enrollments WHERE academic_term_id = $1
			UNION ALL
			SELECT 1 FROM cbc_attendance_periods WHERE academic_term_id = $1
			UNION ALL
			SELECT 1 FROM fee_templates WHERE academic_term_id = $1
			UNION ALL
			SELECT 1 FROM invoices WHERE academic_term_id = $1
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, termID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("academicyears.Repository.HasTermDependents: %w", err)
	}
	return exists, nil
}

// ============================================================================
// SYNC CURRENT TERM (runs inside caller's transaction)
// ============================================================================

// SyncCurrentTerm determines which term should be is_current based on the
// provided "now" date. It runs inside the caller's transaction context.
func (r *PgRepository) SyncCurrentTerm(ctx context.Context, academicYearID string, now time.Time) error {
	// Step 1: Find the term whose date range contains "now"
	const findQuery = `
		SELECT id FROM academic_terms
		WHERE academic_year_id = $1
		  AND deleted_at IS NULL
		  AND start_date <= $2::date
		  AND end_date >= $2::date
		LIMIT 1
	`
	var currentTermID *string
	var tid string
	err := r.pool.QueryRow(ctx, findQuery, academicYearID, now).Scan(&tid)
	if err == nil {
		currentTermID = &tid
	} else if err != pgx.ErrNoRows {
		return fmt.Errorf("academicyears.Repository.SyncCurrentTerm: find: %w", err)
	}

	if currentTermID != nil {
		// Step 2a: Clear is_current on all other terms in this year
		const clearQuery = `
			UPDATE academic_terms
			SET is_current = FALSE, version = version + 1, updated_at = NOW()
			WHERE academic_year_id = $1
			  AND is_current = TRUE
			  AND id != $2
			  AND deleted_at IS NULL
		`
		if _, err := r.pool.Exec(ctx, clearQuery, academicYearID, *currentTermID); err != nil {
			return fmt.Errorf("academicyears.Repository.SyncCurrentTerm: clear others: %w", err)
		}

		// Step 2b: Set is_current on the found term (only if not already current)
		const setQuery = `
			UPDATE academic_terms
			SET is_current = TRUE, version = version + 1, updated_at = NOW()
			WHERE id = $1 AND is_current = FALSE AND deleted_at IS NULL
		`
		if _, err := r.pool.Exec(ctx, setQuery, *currentTermID); err != nil {
			return fmt.Errorf("academicyears.Repository.SyncCurrentTerm: set: %w", err)
		}
	} else {
		// Step 3: No term covers "now" — clear all is_current in this year
		const clearAllQuery = `
			UPDATE academic_terms
			SET is_current = FALSE, version = version + 1, updated_at = NOW()
			WHERE academic_year_id = $1
			  AND is_current = TRUE
			  AND deleted_at IS NULL
		`
		if _, err := r.pool.Exec(ctx, clearAllQuery, academicYearID); err != nil {
			return fmt.Errorf("academicyears.Repository.SyncCurrentTerm: clear all: %w", err)
		}
	}

	return nil
}

// nullableUUID returns a *string for SQL query parameter use. An empty string
// becomes nil (SQL NULL), which the query's $4::uuid IS NULL trick handles.
func nullableUUID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
