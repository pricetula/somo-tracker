package academicyears

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/xerrors"
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

// GetCurrent returns all academic years for a school, with nested
// terms ordered by term_number.
func (r *PgRepository) GetCurrent(ctx context.Context, tenantID, schoolID string) (CurrentAcademicYearWithCurrentTerm, error) {
	const query = `
        SELECT
            ay.id AS academic_year_id,
            ay.name AS academic_year_name,
            ay.start_date AS academic_year_start_date,
            ay.end_date AS academic_year_end_date,
            at.id AS academic_term_id,
            at.name AS academic_term_name,
            at.term_number AS academic_term_number,
            at.start_date AS academic_term_start_date,
            at.end_date AS academic_term_end_date,
            at.is_final AS academic_term_is_final
        FROM academic_years ay
        INNER JOIN academic_terms at ON at.academic_year_id = ay.id
        WHERE ay.tenant_id = $1
          AND ay.school_id = $2
          AND ay.is_current = TRUE
          AND at.is_current = TRUE
        LIMIT 1
    `

	var res CurrentAcademicYearWithCurrentTerm
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID).Scan(
		&res.AcademicYearID,
		&res.AcademicYearName,
		&res.AcademicYearStartDate,
		&res.AcademicYearEndDate,
		&res.AcademicTermID,
		&res.AcademicTermName,
		&res.AcademicTermNumber,
		&res.AcademicTermStartDate,
		&res.AcademicTermEndDate,
		&res.AcademicTermIsFinal,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CurrentAcademicYearWithCurrentTerm{}, fmt.Errorf("academicyears.Repository.GetCurrent: %w", ErrNotFound)
		}
		return CurrentAcademicYearWithCurrentTerm{}, fmt.Errorf("academicyears.Repository.GetCurrent: %w", err)
	}

	return res, nil
}

// ListYears returns all academic years for a school, with nested
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
				) FILTER (WHERE at.id IS NOT NULL),
				'[]'
			) AS terms
		FROM academic_years ay
		LEFT JOIN academic_terms at ON at.academic_year_id = ay.id
		WHERE ay.tenant_id = $1
		  AND ay.school_id = $2
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

// GetYearByID retrieves a single year by primary key.
func (r *PgRepository) GetYearByID(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
	const query = `
		SELECT id, tenant_id, school_id, name,
		       start_date, end_date, is_current,
		       version, created_by, updated_by,
		       created_at, updated_at
		FROM academic_years
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
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

// SetCurrentYear sets is_current = TRUE on a single year. Returns true if a row
// was updated, false otherwise (which translates to 404).
func (r *PgRepository) SetCurrentYear(ctx context.Context, id, tenantID, schoolID, actorID string) (bool, error) {
	const query = `
		UPDATE academic_years
		SET is_current = TRUE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
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

// ListTerms returns all terms, optionally filtered by academic_year_id.
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
		                            term_number, start_date, end_date, is_final,
		                            created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		term.TenantID, term.SchoolID, term.AcademicYearID,
		term.Name, term.TermNumber, term.StartDate, term.EndDate,
		term.IsFinal, term.CreatedBy, term.UpdatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("academicyears.Repository.CreateTerm: %w", mapTermWriteError(err))
	}
	return id, nil
}

// UpdateTerm applies changes to a term, incrementing version. Constraint
// violations raised by the write (GIST exclusion, bounds trigger, date-order
// check) are translated via mapTermWriteError so raw driver strings never
// reach the HTTP layer.
func (r *PgRepository) UpdateTerm(ctx context.Context, term *AcademicTerm) error {
	const query = `
		UPDATE academic_terms
		SET name = $1, start_date = $2, end_date = $3, is_final = $4,
		    version = version + 1, updated_by = $5, updated_at = NOW()
		WHERE id = $6 AND version = $7
	`
	tag, err := r.pool.Exec(ctx, query,
		term.Name, term.StartDate, term.EndDate, term.IsFinal,
		term.UpdatedBy, term.ID, term.Version,
	)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.UpdateTerm: %w", mapTermWriteError(err))
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("academicyears.Repository.UpdateTerm: %w", ErrNotFound)
	}
	return nil
}

// mapTermWriteError translates PostgreSQL constraint violations raised by term
// writes into domain errors:
//   - 23P01 GIST exclusion  → TermDateOverlapError      (409 Conflict)
//   - P0001 bounds trigger  → TermOutOfYearBoundsError  (400 Bad Request)
//   - 23514 date-order check → ErrInvalidInput          (400 Bad Request)
//   - 23505 term_number     → TermNumberExistsError     (409 Conflict)
func mapTermWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23P01": // EXCL_academic_terms_no_overlap
		return &TermDateOverlapError{ConflictingName: "another term"}
	case "P0001": // fn_validate_term_dates_within_year RAISE EXCEPTION
		return &TermOutOfYearBoundsError{}
	case "23514": // chk_term_dates (start_date < end_date)
		return xerrors.InvalidInput("term start_date must be before end_date")
	case "23505": // idx_unique_term_number_per_year
		return &TermNumberExistsError{}
	default:
		return err
	}
}

// DeleteTerm hard-deletes a term.
func (r *PgRepository) DeleteTerm(ctx context.Context, id string) error {
	const query = `
		DELETE FROM academic_terms
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("academicyears.Repository.DeleteTerm: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("academicyears.Repository.DeleteTerm: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// BUSINESS LOGIC CHECKS
// ============================================================================

// FindOverlappingTerms returns terms whose date ranges overlap with the given
// range, optionally excluding a specific term ID (for PATCH self-exclusion).
func (r *PgRepository) FindOverlappingTerms(ctx context.Context, yearID, excludeID string, startDate, endDate time.Time) ([]AcademicTerm, error) {
	const query = `
		SELECT id, name, term_number, start_date, end_date
		FROM academic_terms
		WHERE academic_year_id = $1
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

// termDependencyTables is the set of transactional tables whose rows are
// created against an academic term. These are the tables whose existence
// blocks a hard delete. Derived summary tables (attendance_term_summaries,
// student_term_*_summaries, teacher_*_summaries, class_*_summaries, ...) are
// intentionally excluded — they are recomputed by background workers and are
// safely cascade-deleted.
var termDependencyTables = []struct {
	name   string
	column string
}{
	{"cbc_student_enrollments", "academic_term_id"},
	{"fee_templates", "academic_term_id"},
	{"invoices", "academic_term_id"},
	{"attendance_records", "academic_term_id"},
	{"assessment_sessions", "academic_term_id"},
}

// TermDependencyCounts returns the per-table record counts for every
// transactional table referencing the given term. Callers use this to block
// hard deletion and to report exact dependent resource counts.
func (r *PgRepository) TermDependencyCounts(ctx context.Context, termID string) (map[string]int64, error) {
	var sb strings.Builder
	sb.WriteString("SELECT table_name, cnt FROM (")
	for i, t := range termDependencyTables {
		if i > 0 {
			sb.WriteString(" UNION ALL ")
		}
		fmt.Fprintf(&sb, "SELECT '%s'::text AS table_name, COUNT(*)::bigint AS cnt FROM %s WHERE %s = $1",
			t.name, t.name, t.column)
	}
	sb.WriteString(") dep GROUP BY table_name, cnt")

	rows, err := r.pool.Query(ctx, sb.String(), termID)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.TermDependencyCounts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64, len(termDependencyTables))
	for rows.Next() {
		var name string
		var cnt int64
		if err := rows.Scan(&name, &cnt); err != nil {
			return nil, fmt.Errorf("academicyears.Repository.TermDependencyCounts: scan: %w", err)
		}
		counts[name] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.TermDependencyCounts: rows: %w", err)
	}
	return counts, nil
}

// CountOrphansOutsideRange counts assessment sessions and attendance records
// linked to the term whose recorded date falls OUTSIDE a proposed new date
// range. Used to guard against narrowing an active term's dates in a way that
// strands existing transactional data.
func (r *PgRepository) CountOrphansOutsideRange(ctx context.Context, termID string, newStart, newEnd time.Time) (map[string]int64, error) {
	const query = `
		SELECT 'assessment_sessions', COUNT(*)::bigint
		FROM assessment_sessions
		WHERE academic_term_id = $1
		  AND scheduled_date IS NOT NULL
		  AND (scheduled_date < $2::date OR scheduled_date > $3::date)
		UNION ALL
		SELECT 'attendance_records', COUNT(*)::bigint
		FROM attendance_records
		WHERE academic_term_id = $1
		  AND (date < $2::date OR date > $3::date)
	`
	rows, err := r.pool.Query(ctx, query, termID, newStart, newEnd)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.CountOrphansOutsideRange: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64, 2)
	for rows.Next() {
		var name string
		var cnt int64
		if err := rows.Scan(&name, &cnt); err != nil {
			return nil, fmt.Errorf("academicyears.Repository.CountOrphansOutsideRange: scan: %w", err)
		}
		counts[name] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.CountOrphansOutsideRange: rows: %w", err)
	}
	return counts, nil
}

// ============================================================================
// ACTIVATE TERM (atomic, app-level transaction)
// ============================================================================

// ActivateTerm atomically makes a single term — and its parent academic year —
// the school's current term/year, deactivating whatever is currently active.
//
// Locking order (deadlock-free):
//  1. target term   — SELECT ... FOR UPDATE (scoped by tenant + school)
//  2. current term  — SELECT ... FOR UPDATE (school-wide is_current = TRUE)
//  3. years         — SELECT ... FOR UPDATE ORDER BY id (target year, then
//     current year)
//
// idx_one_current_term_per_school guarantees at most one current term per
// school, so the current-term row is the serialization point: every activation
// locks its own target term first, then contends on the same current-term row
// before touching any year rows. All activation paths acquire locks in this
// order, so no lock-ordering deadlock is possible.
//
// All affected rows (old current term, old current year, target term, target
// year) get version + 1, updated_by = actorID and updated_at = NOW().
func (r *PgRepository) ActivateTerm(ctx context.Context, termID, tenantID, schoolID, actorID string) (*AcademicTerm, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: begin: %w", err)
	}
	defer func() {
		// Rollback is a no-op after a successful Commit (pgx.ErrTxClosed); on
		// failure it aborts the transaction before the pool recycles the conn.
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock the target term and confirm it belongs to this tenant + school.
	var targetYearID string
	const lockTargetQuery = `
		SELECT id, academic_year_id
		FROM academic_terms
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		FOR UPDATE
	`
	if err := tx.QueryRow(ctx, lockTargetQuery, termID, tenantID, schoolID).Scan(&termID, &targetYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: lock target: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: lock target: %w", err)
	}

	// 2. Lock the school's currently-active term (if any). This is the
	//    serialization point for concurrent activations.
	var currentTermID, currentYearID string
	const lockCurrentQuery = `
		SELECT at.id, at.academic_year_id
		FROM academic_terms at
		JOIN academic_years ay ON ay.id = at.academic_year_id
		WHERE at.tenant_id = $1 AND at.school_id = $2 AND at.is_current = TRUE
		  AND ay.tenant_id = $1 AND ay.school_id = $2
		FOR UPDATE OF at
	`
	hasCurrent := true
	if err := tx.QueryRow(ctx, lockCurrentQuery, tenantID, schoolID).Scan(&currentTermID, &currentYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			hasCurrent = false
		} else {
			return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: lock current: %w", err)
		}
	}

	// 3. Lock the affected years in deterministic (id) order. Idempotent case:
	//    activating the already-current term touches only the target year.
	yearIDs := []string{targetYearID}
	if hasCurrent && currentYearID != targetYearID {
		yearIDs = append(yearIDs, currentYearID)
	}
	const lockYearsQuery = `
		SELECT id
		FROM academic_years
		WHERE id = ANY($1::uuid[]) AND tenant_id = $2 AND school_id = $3
		ORDER BY id
		FOR UPDATE
	`
	if _, err := tx.Exec(ctx, lockYearsQuery, yearIDs, tenantID, schoolID); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: lock years: %w", err)
	}

	// 4a. Deactivate the old current term (idempotent when target is current).
	const clearTermQuery = `
		UPDATE academic_terms
		SET is_current = FALSE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE tenant_id = $1 AND school_id = $2 AND is_current = TRUE AND id != $3
	`
	if _, err := tx.Exec(ctx, clearTermQuery, tenantID, schoolID, termID, actorID); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: clear current term: %w", err)
	}

	// 4b. Deactivate the old current year (idempotent when target is current).
	const clearYearQuery = `
		UPDATE academic_years
		SET is_current = FALSE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE tenant_id = $1 AND school_id = $2 AND is_current = TRUE AND id != $3
	`
	if _, err := tx.Exec(ctx, clearYearQuery, tenantID, schoolID, targetYearID, actorID); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: clear current year: %w", err)
	}

	// 5a. Activate the target term, returning the fresh row for the response.
	const setTermQuery = `
		UPDATE academic_terms
		SET is_current = TRUE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		RETURNING id, academic_year_id, name, term_number, start_date, end_date,
		          is_current, is_final, version, created_by, updated_by,
		          created_at, updated_at
	`
	var activated AcademicTerm
	if err := tx.QueryRow(ctx, setTermQuery, termID, tenantID, schoolID, actorID).Scan(
		&activated.ID, &activated.AcademicYearID, &activated.Name, &activated.TermNumber,
		&activated.StartDate, &activated.EndDate, &activated.IsCurrent, &activated.IsFinal,
		&activated.Version, &activated.CreatedBy, &activated.UpdatedBy,
		&activated.CreatedAt, &activated.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: set target term: %w", err)
	}

	// 5b. Activate the target term's parent year.
	const setYearQuery = `
		UPDATE academic_years
		SET is_current = TRUE, version = version + 1, updated_by = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	if _, err := tx.Exec(ctx, setYearQuery, targetYearID, tenantID, schoolID, actorID); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: set target year: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("academicyears.Repository.ActivateTerm: commit: %w", err)
	}

	return &activated, nil
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
		`
		if _, err := r.pool.Exec(ctx, clearQuery, academicYearID, *currentTermID); err != nil {
			return fmt.Errorf("academicyears.Repository.SyncCurrentTerm: clear others: %w", err)
		}

		// Step 2b: Set is_current on the found term (only if not already current)
		const setQuery = `
			UPDATE academic_terms
			SET is_current = TRUE, version = version + 1, updated_at = NOW()
			WHERE id = $1 AND is_current = FALSE
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
		`
		if _, err := r.pool.Exec(ctx, clearAllQuery, academicYearID); err != nil {
			return fmt.Errorf("academicyears.Repository.SyncCurrentTerm: clear all: %w", err)
		}
	}

	return nil
}

// ============================================================================
// Current Academic Year / Term Lookups
// ============================================================================

// GetCurrentAcademicYearID returns the ID of the current academic year for the school.
func (r *PgRepository) GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error) {
	const query = `
		SELECT id FROM academic_years
		WHERE tenant_id = $1 AND school_id = $2 AND is_current = TRUE
		LIMIT 1
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("academicyears.Repository.GetCurrentAcademicYearID: %w", err)
	}
	return id, nil
}

// GetCurrentAcademicTermID returns the ID of the current active term for the given academic year.
func (r *PgRepository) GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error) {
	const query = `
		SELECT id FROM academic_terms
		WHERE academic_year_id = $1 AND is_current = TRUE
		LIMIT 1
	`
	var id string
	err := r.pool.QueryRow(ctx, query, academicYearID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("academicyears.Repository.GetCurrentAcademicTermID: %w", err)
	}
	return id, nil
}

// nullableUUID returns a *string for SQL query parameter use. An empty string
// becomes nil (SQL NULL), which the query's $4::uuid IS NULL trick handles.
func nullableUUID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
