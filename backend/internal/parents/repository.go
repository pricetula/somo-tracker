package parents

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// PgRepository handles parent database operations.
type PgRepository struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools, logger *zap.SugaredLogger) *PgRepository {
	return &PgRepository{pool: pools.PG, logger: logger}
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

func joinORs(elems []string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for i := 1; i < len(elems); i++ {
		result += " OR " + elems[i]
	}
	return result
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
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, studentID, tenantID).Scan(&exists)
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
	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return "", fmt.Errorf("parents.Repository.Create: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			r.logger.Warnw("parents.Repository.Create: rollback",
				"error", rbErr.Error())
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

// parentJoinColumns is the common SELECT list for membership-based parent queries.
// Queries from memberships (role = 'PARENT') with LEFT JOIN to cbc_parents so that
// parents without a student link (no cbc_parents record) are still found.
const parentJoinColumns = `
	COALESCE(cp.id::text, u.id::text),
	m.tenant_id,
	m.user_id,
	u.full_name,
	u.email,
	COALESCE(cp.phone_number, ''),
	COALESCE(cp.is_active, u.is_active),
	COALESCE(cp.created_at::text, u.created_at::text)
`

// parentJoin is the common FROM/JOIN clause for membership-based parent queries.
const parentJoin = `
	FROM memberships m
	JOIN users u ON u.id = m.user_id AND u.tenant_id = m.tenant_id
	LEFT JOIN cbc_parents cp ON cp.user_id = m.user_id AND cp.tenant_id = m.tenant_id
`

// parentRoleFilter is the role filter clause appended to every parent query.
const parentRoleFilter = `m.role = 'PARENT'`

// ============================================================================
// READ
// ============================================================================

// GetByUserID retrieves a parent profile by the linked user_id.
func (r *PgRepository) GetByUserID(ctx context.Context, userID, tenantID string) (*Parent, error) {
	const query = `SELECT ` + parentJoinColumns + parentJoin + ` WHERE ` + parentRoleFilter + ` AND m.user_id = $1 AND m.tenant_id = $2`
	p, err := scanParent(database.FromContext(ctx, r.pool).QueryRow(ctx, query, userID, tenantID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("parents.Repository.GetByUserID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("parents.Repository.GetByUserID: %w", err)
	}
	return p, nil
}

// GetByID retrieves a single parent by primary key.
// The id may be either a cbc_parents.id (for linked parents) or a users.id
// (for parents without a student link). Both are accepted.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID string) (*Parent, error) {
	const query = `SELECT ` + parentJoinColumns + parentJoin + ` WHERE ` + parentRoleFilter + ` AND m.tenant_id = $2 AND (cp.id::text = $1 OR u.id::text = $1)`
	p, err := scanParent(database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID))
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
		  AND sp.tenant_id = $2
		ORDER BY sp.is_primary DESC, s.full_name ASC
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, studentsQuery, id, tenantID)
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

// List returns parents optionally filtered by search (name/email), student_id,
// or curriculum filters (education_level, grade_level), with pagination.
// The base source is the memberships table filtered to role = 'PARENT', so that
// parents who have not been linked to students (no cbc_parents record) are also
// included in the listing.
func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Parent, int, error) {
	whereClause := `WHERE ` + parentRoleFilter + ` AND m.tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.Search != "" {
		whereClause += fmt.Sprintf(` AND (u.full_name ILIKE $%d OR u.email ILIKE $%d)`, argIdx, argIdx+1)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIdx += 2
	}

	if filter.StudentID != "" {
		whereClause += fmt.Sprintf(` AND cp.id IN (
			SELECT sp.parent_id FROM cbc_student_parents sp
			WHERE sp.student_id = $%d
		)`, argIdx)
		args = append(args, filter.StudentID)
		argIdx++
	}

	// ── Education Level multi-select ────────────────────────────────────
	if len(filter.EducationLevels) > 0 {
		ors := make([]string, len(filter.EducationLevels))
		for i, el := range filter.EducationLevels {
			ors[i] = fmt.Sprintf("la.education_level::text = $%d", argIdx)
			args = append(args, el)
			argIdx++
		}
		whereClause += ` AND cp.id IN (
			SELECT sp.parent_id
			FROM cbc_student_parents sp
			JOIN cbc_student_enrollments se ON se.student_id = sp.student_id
			JOIN cbc_classes cc ON cc.id = se.class_id
			JOIN cbc_learning_areas la ON la.grade_level = cc.grade_level
			WHERE ` + joinORs(ors) + `
		)`
	}

	// ── Grade Level multi-select ────────────────────────────────────────
	if len(filter.GradeLevels) > 0 {
		ors := make([]string, len(filter.GradeLevels))
		for i, gl := range filter.GradeLevels {
			ors[i] = fmt.Sprintf("cc.grade_level::text = $%d", argIdx)
			args = append(args, gl)
			argIdx++
		}
		whereClause += ` AND cp.id IN (
			SELECT sp.parent_id
			FROM cbc_student_parents sp
			JOIN cbc_student_enrollments se ON se.student_id = sp.student_id
			JOIN cbc_classes cc ON cc.id = se.class_id
			WHERE ` + joinORs(ors) + `
		)`
	}

	// Count query
	countQuery := `SELECT COUNT(*)` + parentJoin + ` ` + whereClause
	var total int
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("parents.Repository.List: count: %w", err)
	}

	if total == 0 {
		return []Parent{}, 0, nil
	}

	offset := (filter.Page - 1) * filter.Limit
	dataQuery := `SELECT ` + parentJoinColumns + parentJoin + ` ` + whereClause + ` ORDER BY u.full_name ASC LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)
	dataArgs := append(args, filter.Limit, offset)

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("parents.Repository.List: query: %w", err)
	}
	defer rows.Close()

	parents, err := scanParentsRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return parents, total, nil
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

	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, args...)
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
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID)
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
	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("parents.Repository.LinkStudent: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			r.logger.Warnw("parents.Repository.LinkStudent: rollback",
				"error", rbErr.Error())
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
			WHERE student_id = $1 AND tenant_id = $2 AND is_primary = true
		`
		_, err = tx.Exec(ctx, demoteQuery, payload.StudentID, tenantID)
		if err != nil {
			return fmt.Errorf("parents.Repository.LinkStudent: demote: %w", err)
		}
	}

	// Insert junction row
	const linkQuery = `
		INSERT INTO cbc_student_parents (tenant_id, student_id, parent_id, relationship, is_primary)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, linkQuery,
		tenantID, payload.StudentID, parentID, payload.Relationship, isPrimary,
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
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, parentID, studentID, tenantID)
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
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, query, studentID, tenantID)
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
		WHERE sp.student_id = $1
		  AND sp.tenant_id = $2
		  AND cp.tenant_id = $2
	`
	var count int
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, studentID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("parents.Repository.CountLinksByStudent: %w", err)
	}
	return count, nil
}

// GetStytchOrgID retrieves the Stytch organization ID for a tenant.
func (r *PgRepository) GetStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	const query = `SELECT stytch_org_id FROM tenants WHERE id = $1`
	var orgID string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID).Scan(&orgID)
	if err != nil {
		return "", fmt.Errorf("parents.Repository.GetStytchOrgID: %w", err)
	}
	return orgID, nil
}
