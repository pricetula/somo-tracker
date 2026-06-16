package academiccalendar

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

const dateFmt = "2006-01-02"

// GetPrimarySchoolID returns the primary active school for a tenant.
// If the user has a membership, prefers that school; otherwise picks the first active school.
func (r *Repository) GetPrimarySchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	// Try membership-first: get the school the user belongs to
	const membershipQuery = `
		SELECT school_id FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		LIMIT 1
	`
	var schoolID string
	err := r.pool.QueryRow(ctx, membershipQuery, tenantID, userID).Scan(&schoolID)
	if err == nil {
		return schoolID, nil
	}

	// Fallback: first active school for the tenant
	const fallbackQuery = `
		SELECT id FROM schools
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at ASC
		LIMIT 1
	`
	err = r.pool.QueryRow(ctx, fallbackQuery, tenantID).Scan(&schoolID)
	if err != nil {
		return "", fmt.Errorf("no active school found for tenant %s: %w", tenantID, err)
	}
	return schoolID, nil
}

// Repository handles database operations for the academic calendar.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pools *database.Pools) *Repository {
	return &Repository{pool: pools.PG}
}

// yearRow is the intermediate scan target for an academic year row.
type yearRow struct {
	ID        string
	Name      string
	StartDate time.Time
	EndDate   time.Time
}

// termRow is the intermediate scan target for a term row.
type termRow struct {
	ID        string
	YearID    string
	Name      string
	StartDate time.Time
	EndDate   time.Time
	IsFinal   bool
}

// ---------------------------------------------------------------------------
// Query: GetCurrentCalendar
// ---------------------------------------------------------------------------

// GetCurrentCalendar fetches the current academic year + its terms for a school.
// Returns nil if no current year is set.
func (r *Repository) GetCurrentCalendar(ctx context.Context, schoolID, tenantID string) (*AcademicYear, error) {
	yr, err := r.getCurrentYearRow(ctx, schoolID, tenantID)
	if err != nil {
		return nil, err
	}
	if yr == nil {
		return nil, nil
	}

	terms, err := r.getYearTerms(ctx, yr.ID, tenantID)
	if err != nil {
		return nil, err
	}

	yearNum, _ := strconv.Atoi(yr.Name)

	periods := make([]AcademicPeriod, 0, len(terms))
	for _, t := range terms {
		periods = append(periods, AcademicPeriod{
			ID:        t.ID,
			Name:      t.Name,
			StartDate: t.StartDate.Format(dateFmt),
			EndDate:   t.EndDate.Format(dateFmt),
			IsFinal:   t.IsFinal,
		})
	}

	return &AcademicYear{
		ID:      yr.ID,
		Year:    yearNum,
		Periods: periods,
	}, nil
}

func (r *Repository) getCurrentYearRow(ctx context.Context, schoolID, tenantID string) (*yearRow, error) {
	const query = `
		SELECT id, name, start_date, end_date
		FROM academic_years
		WHERE school_id = $1 AND tenant_id = $2 AND is_current = true
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, schoolID, tenantID)

	var yr yearRow
	err := row.Scan(&yr.ID, &yr.Name, &yr.StartDate, &yr.EndDate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get current academic year: %w", err)
	}
	return &yr, nil
}

func (r *Repository) getYearTerms(ctx context.Context, yearID, tenantID string) ([]termRow, error) {
	const query = `
		SELECT id, academic_year_id, name, start_date, end_date, is_final
		FROM academic_terms
		WHERE academic_year_id = $1 AND tenant_id = $2
		ORDER BY start_date ASC
	`
	rows, err := r.pool.Query(ctx, query, yearID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get terms for year %s: %w", yearID, err)
	}
	defer rows.Close()

	var terms []termRow
	for rows.Next() {
		var t termRow
		if err := rows.Scan(&t.ID, &t.YearID, &t.Name, &t.StartDate, &t.EndDate, &t.IsFinal); err != nil {
			return nil, fmt.Errorf("scan term: %w", err)
		}
		terms = append(terms, t)
	}
	return terms, nil
}

// ---------------------------------------------------------------------------
// Transactional save helpers
// ---------------------------------------------------------------------------

// BeginTx starts a transaction.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// UnsetCurrentYears marks all academic years for this school as non-current.
func (r *Repository) UnsetCurrentYears(ctx context.Context, tx pgx.Tx, schoolID, tenantID string) error {
	const query = `UPDATE academic_years SET is_current = false WHERE school_id = $1 AND tenant_id = $2`
	_, err := tx.Exec(ctx, query, schoolID, tenantID)
	return err
}

// UpsertYear creates or updates the current academic year, returning its ID.
func (r *Repository) UpsertYear(ctx context.Context, tx pgx.Tx, schoolID, tenantID string, year int, startDate, endDate string) (string, error) {
	yearName := strconv.Itoa(year)

	// Look for existing year with this name for this school
	const lookupQuery = `
		SELECT id FROM academic_years
		WHERE school_id = $1 AND tenant_id = $2 AND name = $3
		LIMIT 1
	`
	var existingID string
	err := tx.QueryRow(ctx, lookupQuery, schoolID, tenantID, yearName).Scan(&existingID)

	if err == nil {
		// Update existing — set as current
		const updateQuery = `
			UPDATE academic_years
			SET start_date = $1, end_date = $2, is_current = true
			WHERE id = $3
			RETURNING id
		`
		if err := tx.QueryRow(ctx, updateQuery, startDate, endDate, existingID).Scan(&existingID); err != nil {
			return "", fmt.Errorf("update academic year: %w", err)
		}
		return existingID, nil
	}

	if err != pgx.ErrNoRows {
		return "", fmt.Errorf("lookup academic year: %w", err)
	}

	// Insert new year as current
	const insertQuery = `
		INSERT INTO academic_years (tenant_id, school_id, name, start_date, end_date, is_current)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id
	`
	if err := tx.QueryRow(ctx, insertQuery, tenantID, schoolID, yearName, startDate, endDate).Scan(&existingID); err != nil {
		return "", fmt.Errorf("insert academic year: %w", err)
	}
	return existingID, nil
}

// ReplaceTerms deletes all existing terms for a given year and inserts new ones.
func (r *Repository) ReplaceTerms(ctx context.Context, tx pgx.Tx, yearID, tenantID string, periods []SavePeriodPayload) error {
	const deleteQuery = `DELETE FROM academic_terms WHERE academic_year_id = $1 AND tenant_id = $2`
	if _, err := tx.Exec(ctx, deleteQuery, yearID, tenantID); err != nil {
		return fmt.Errorf("delete existing terms: %w", err)
	}

	const insertQuery = `
		INSERT INTO academic_terms (tenant_id, academic_year_id, name, start_date, end_date, is_final)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	for _, p := range periods {
		if _, err := tx.Exec(ctx, insertQuery, tenantID, yearID, p.Name, p.StartDate, p.EndDate, p.IsFinal); err != nil {
			return fmt.Errorf("insert term %q: %w", p.Name, err)
		}
	}
	return nil
}
