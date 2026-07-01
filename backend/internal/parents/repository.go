package parents

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles parent database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// isUniqueViolation checks if an error is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	msg := err.Error()
	return contains(msg, "unique constraint") || contains(msg, "duplicate key")
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

// ============================================================================
// Cross-Domain Resolver: StudentResolver
// ============================================================================

// StudentExistsInTenant checks whether a student exists and belongs to
// the given tenant.
func (r *PgRepository) StudentExistsInTenant(ctx context.Context, studentID, tenantID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_students
			WHERE id = $1 AND tenant_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, studentID, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("parents.Repository.StudentExistsInTenant: %w", err)
	}
	return exists, nil
}

// ============================================================================
// CREATE
// ============================================================================

// Create inserts a new parent profile, creating a platform user if one
// doesn't already exist with the given email in the tenant.
func (r *PgRepository) Create(ctx context.Context, tenantID string, payload CreateParentPayload) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("parents.Repository.Create: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			_ = err
		}
	}()

	// Find or create the user
	const findUserQuery = `
		SELECT id FROM users
		WHERE email = $1 AND tenant_id = $2
	`
	var userID string
	err = tx.QueryRow(ctx, findUserQuery, payload.Email, tenantID).Scan(&userID)
	if err != nil {
		if err != pgx.ErrNoRows {
			return "", fmt.Errorf("parents.Repository.Create: find user: %w", err)
		}

		// Create new user
		const createUserQuery = `
			INSERT INTO users (email, tenant_id, full_name, is_active)
			VALUES ($1, $2, $3, true)
			RETURNING id
		`
		err = tx.QueryRow(ctx, createUserQuery,
			payload.Email, tenantID, payload.FullName,
		).Scan(&userID)
		if err != nil {
			if isUniqueViolation(err) {
				return "", fmt.Errorf("parents.Repository.Create: create user: %w", ErrAlreadyExists)
			}
			return "", fmt.Errorf("parents.Repository.Create: create user: %w", err)
		}
	}

	// Create parent profile
	const createParentQuery = `
		INSERT INTO cbc_parents (tenant_id, user_id, phone_number)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var parentID string
	err = tx.QueryRow(ctx, createParentQuery, tenantID, userID, payload.PhoneNumber).Scan(&parentID)
	if err != nil {
		if isUniqueViolation(err) {
			return "", fmt.Errorf("parents.Repository.Create: parent profile: %w", ErrAlreadyExists)
		}
		return "", fmt.Errorf("parents.Repository.Create: parent profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("parents.Repository.Create: commit: %w", err)
	}

	return parentID, nil
}

// ============================================================================
// READ
// ============================================================================

// scanParent scans a single Parent row from the cbc_parents + users join.
func scanParent(row pgx.Row) (*Parent, error) {
	var p Parent
	err := row.Scan(
		&p.ID, &p.TenantID, &p.UserID,
		&p.FullName, &p.Email, &p.PhoneNumber,
		&p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// scanParentsRows scans all rows from a result set into a slice.
func scanParentsRows(rows pgx.Rows) ([]Parent, error) {
	var parents []Parent
	for rows.Next() {
		var p Parent
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.UserID,
			&p.FullName, &p.Email, &p.PhoneNumber,
			&p.IsActive, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("parents.Repository.scanParentsRows: scan: %w", err)
		}
		parents = append(parents, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parents.Repository.scanParentsRows: rows: %w", err)
	}
	if parents == nil {
		parents = []Parent{}
	}
	return parents, nil
}

// parentJoinColumns is the common SELECT list for joining cbc_parents + users.
const parentJoinColumns = `
	cp.id, cp.tenant_id, cp.user_id,
	u.full_name, u.email, cp.phone_number,
	cp.is_active, cp.created_at::text
`

// parentJoin is the common FROM/JOIN clause.
const parentJoin = `
	FROM cbc_parents cp
	JOIN users u ON u.id = cp.user_id AND u.tenant_id = cp.tenant_id
`

// GetByID retrieves a single parent by primary key.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID string) (*Parent, error) {
	const query = `SELECT ` + parentJoinColumns + parentJoin + ` WHERE cp.id = $1 AND cp.tenant_id = $2`
	p, err := scanParent(r.pool.QueryRow(ctx, query, id, tenantID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("parents.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("parents.Repository.GetByID: %w", err)
	}
	return p, nil
}

// GetDetail retrieves a parent with all linked students.
func (r *PgRepository) GetDetail(ctx context.Context, id, tenantID string) (*ParentDetail, error) {
	// First fetch the parent
	p, err := r.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("parents.Repository.GetDetail: %w", err)
	}

	// Then fetch linked students
	const studentsQuery = `
		SELECT sp.student_id, s.full_name, sp.relationship, sp.is_primary
		FROM cbc_student_parents sp
		JOIN cbc_students s ON s.id = sp.student_id AND s.tenant_id = $2
		WHERE sp.parent_id = $1
		ORDER BY sp.is_primary DESC, s.full_name ASC
	`
	rows, err := r.pool.Query(ctx, studentsQuery, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("parents.Repository.GetDetail: linked students: %w", err)
	}
	defer rows.Close()

	var links []StudentLink
	for rows.Next() {
		var sl StudentLink
		if err := rows.Scan(&sl.StudentID, &sl.FullName, &sl.Relationship, &sl.IsPrimary); err != nil {
			return nil, fmt.Errorf("parents.Repository.GetDetail: scan link: %w", err)
		}
		links = append(links, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parents.Repository.GetDetail: rows: %w", err)
	}
	if links == nil {
		links = []StudentLink{}
	}

	return &ParentDetail{
		Parent:         *p,
		LinkedStudents: links,
	}, nil
}

// List returns parents optionally filtered by search (name/email) or student_id.
func (r *PgRepository) List(ctx context.Context, tenantID string, search, studentID string) ([]Parent, error) {
	baseQuery := `SELECT ` + parentJoinColumns + parentJoin + ` WHERE cp.tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if search != "" {
		baseQuery += fmt.Sprintf(` AND (u.full_name ILIKE $%d OR u.email ILIKE $%d)`, argIdx, argIdx+1)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
		argIdx += 2
	}

	if studentID != "" {
		baseQuery += fmt.Sprintf(` AND cp.id IN (
			SELECT sp.parent_id FROM cbc_student_parents sp
			WHERE sp.student_id = $%d
		)`, argIdx)
		args = append(args, studentID)
	}

	baseQuery += ` ORDER BY u.full_name ASC`

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("parents.Repository.List: %w", err)
	}
	defer rows.Close()

	return scanParentsRows(rows)
}

// ============================================================================
// UPDATE
// ============================================================================

// Update applies partial updates to a parent profile.
func (r *PgRepository) Update(ctx context.Context, id, tenantID string, payload UpdateParentPayload) error {
	// Build dynamic UPDATE
	query := `UPDATE cbc_parents SET updated_at = NOW()`
	args := []interface{}{}
	argIdx := 1

	if payload.PhoneNumber != nil {
		query += fmt.Sprintf(", phone_number = $%d", argIdx)
		args = append(args, *payload.PhoneNumber)
		argIdx++
	}
	if payload.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *payload.IsActive)
		argIdx++
	}

	// No fields to update
	if len(args) == 0 {
		return fmt.Errorf("parents.Repository.Update: %w", ErrInvalidInput)
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argIdx, argIdx+1)
	args = append(args, id, tenantID)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("parents.Repository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("parents.Repository.Update: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// Delete removes a parent profile. The linked user record is preserved.
// Foreign key SET NULL / CASCADE behavior handles invoice references.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID string) error {
	const query = `DELETE FROM cbc_parents WHERE id = $1 AND tenant_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("parents.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("parents.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// STUDENT LINKING
// ============================================================================

// LinkStudent links a student to a parent.
func (r *PgRepository) LinkStudent(ctx context.Context, parentID, tenantID string, payload LinkStudentPayload) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("parents.Repository.LinkStudent: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			_ = err
		}
	}()

	// Validate student belongs to tenant
	const checkStudentQuery = `
		SELECT EXISTS (SELECT 1 FROM cbc_students WHERE id = $1 AND tenant_id = $2)
	`
	var studentExists bool
	err = tx.QueryRow(ctx, checkStudentQuery, payload.StudentID, tenantID).Scan(&studentExists)
	if err != nil {
		return fmt.Errorf("parents.Repository.LinkStudent: check student: %w", err)
	}
	if !studentExists {
		return fmt.Errorf("parents.Repository.LinkStudent: %w", ErrStudentNotFound)
	}

	// If is_primary is true, demote all existing primary links for this student
	isPrimary := false
	if payload.IsPrimary != nil {
		isPrimary = *payload.IsPrimary
	}
	if isPrimary {
		const demoteQuery = `
			UPDATE cbc_student_parents
			SET is_primary = false
			WHERE student_id = $1 AND is_primary = true
		`
		_, err = tx.Exec(ctx, demoteQuery, payload.StudentID)
		if err != nil {
			return fmt.Errorf("parents.Repository.LinkStudent: demote: %w", err)
		}
	}

	// Insert junction row
	const linkQuery = `
		INSERT INTO cbc_student_parents (student_id, parent_id, relationship, is_primary)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, linkQuery,
		payload.StudentID, parentID, payload.Relationship, isPrimary,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("parents.Repository.LinkStudent: %w", ErrDuplicateLink)
		}
		return fmt.Errorf("parents.Repository.LinkStudent: insert link: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("parents.Repository.LinkStudent: commit: %w", err)
	}

	return nil
}

// UnlinkStudent removes a student-parent link.
func (r *PgRepository) UnlinkStudent(ctx context.Context, parentID, studentID, tenantID string) error {
	const query = `
		DELETE FROM cbc_student_parents sp
		USING cbc_parents cp
		WHERE cp.id = sp.parent_id
		  AND sp.parent_id = $1
		  AND sp.student_id = $2
		  AND cp.tenant_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, parentID, studentID, tenantID)
	if err != nil {
		return fmt.Errorf("parents.Repository.UnlinkStudent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("parents.Repository.UnlinkStudent: %w", ErrNotFound)
	}
	return nil
}

// DemotePrimaryForStudent clears the is_primary flag for all parents linked
// to the given student within the tenant.
func (r *PgRepository) DemotePrimaryForStudent(ctx context.Context, studentID, tenantID string) error {
	const query = `
		UPDATE cbc_student_parents sp
		SET is_primary = false
		FROM cbc_parents cp
		WHERE cp.id = sp.parent_id
		  AND sp.student_id = $1
		  AND cp.tenant_id = $2
		  AND sp.is_primary = true
	`
	_, err := r.pool.Exec(ctx, query, studentID, tenantID)
	if err != nil {
		return fmt.Errorf("parents.Repository.DemotePrimaryForStudent: %w", err)
	}
	return nil
}

// CountLinksByStudent returns the number of parents linked to a student.
func (r *PgRepository) CountLinksByStudent(ctx context.Context, studentID, tenantID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM cbc_student_parents sp
		JOIN cbc_parents cp ON cp.id = sp.parent_id
		WHERE sp.student_id = $1 AND cp.tenant_id = $2
	`
	var count int
	err := r.pool.QueryRow(ctx, query, studentID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("parents.Repository.CountLinksByStudent: %w", err)
	}
	return count, nil
}
