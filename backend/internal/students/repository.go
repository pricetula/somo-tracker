package students

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository handles student database operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pools *database.Pools) *Repository {
	return &Repository{pool: pools.PG}
}

// ListByTenant returns paginated students for a tenant with optional search.
func (r *Repository) ListByTenant(ctx context.Context, tenantID string, offset, limit int, search string) ([]Student, int, error) {
	// Count total matching rows
	countQuery := `SELECT COUNT(*) FROM students WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}

	if search != "" {
		countQuery += ` AND (first_name ILIKE $2 OR last_name ILIKE $3 OR middle_name ILIKE $4)`
		pattern := "%" + search + "%"
		countArgs = append(countArgs, pattern, pattern, pattern)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count students: %w", err)
	}

	// Fetch paginated rows
	dataQuery := `
		SELECT id, tenant_id, first_name, middle_name, last_name, gender, date_of_birth::text, is_active, created_at
		FROM students
		WHERE tenant_id = $1`

	dataArgs := []interface{}{tenantID}
	nextArg := 2

	if search != "" {
		dataQuery += fmt.Sprintf(` AND (first_name ILIKE $%d OR last_name ILIKE $%d OR middle_name ILIKE $%d)`, nextArg, nextArg+1, nextArg+2)
		pattern := "%" + search + "%"
		dataArgs = append(dataArgs, pattern, pattern, pattern)
		nextArg += 3
	}

	dataQuery += ` ORDER BY created_at DESC`
	dataQuery += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, nextArg, nextArg+1)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.ID, &s.TenantID, &s.FirstName, &s.MiddleName, &s.LastName, &s.Gender, &s.DateOfBirth, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan student: %w", err)
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if students == nil {
		students = []Student{}
	}

	return students, total, nil
}

// Create inserts a single student record and returns it.
func (r *Repository) Create(ctx context.Context, tenantID string, payload CreateStudentPayload) (*Student, error) {
	const query = `
		INSERT INTO students (tenant_id, first_name, middle_name, last_name, gender, date_of_birth)
		VALUES ($1, $2, $3, $4, $5, $6::date)
		RETURNING id, tenant_id, first_name, middle_name, last_name, gender, date_of_birth::text, is_active, created_at
	`

	var s Student
	err := r.pool.QueryRow(ctx, query,
		tenantID,
		payload.FirstName,
		payload.MiddleName,
		payload.LastName,
		payload.Gender,
		payload.DateOfBirth,
	).Scan(&s.ID, &s.TenantID, &s.FirstName, &s.MiddleName, &s.LastName, &s.Gender, &s.DateOfBirth, &s.IsActive, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert student: %w", err)
	}
	return &s, nil
}

// BulkInsert inserts multiple students in a single batch.
// Returns the count of successfully inserted rows.
func (r *Repository) BulkInsert(ctx context.Context, tenantID string, rows []CSVRawRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Batch insert using pgx's CopyFrom for maximum performance
	// Map into [][]interface{}
	input := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		middleName := &row.MiddleName
		if row.MiddleName == "" {
			middleName = nil
		}
		input = append(input, []interface{}{
			tenantID,
			row.FirstName,
			middleName,
			row.LastName,
			row.Gender,
			row.DateOfBirth,
		})
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"students"},
		[]string{"tenant_id", "first_name", "middle_name", "last_name", "gender", "date_of_birth"},
		pgx.CopyFromRows(input),
	)
	if err != nil {
		return 0, fmt.Errorf("copy from: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return len(rows), nil
}
