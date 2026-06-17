package school

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// SqlcRepository is the concrete PostgreSQL-backed implementation.
type SqlcRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new SqlcRepository.
func NewRepository(pools *database.Pools) *SqlcRepository {
	return &SqlcRepository{pool: pools.PG}
}

// ListByTenant returns all active schools for a tenant.
func (r *SqlcRepository) ListByTenant(ctx context.Context, tenantID string) ([]School, error) {
	const query = `
		SELECT id, tenant_id, education_system_id, name, is_active, is_demo
		FROM schools
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list schools by tenant: %w", err)
	}
	defer rows.Close()

	var schools []School
	for rows.Next() {
		var s School
		if err := rows.Scan(&s.ID, &s.TenantID, &s.EducationSystemID, &s.Name, &s.IsActive, &s.IsDemo); err != nil {
			return nil, fmt.Errorf("scan school: %w", err)
		}
		schools = append(schools, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if schools == nil {
		schools = []School{}
	}
	return schools, nil
}

// GetByID retrieves a school by ID.
func (r *SqlcRepository) GetByID(ctx context.Context, id string) (*School, error) {
	const query = `
		SELECT id, tenant_id, education_system_id, name, is_active, is_demo
		FROM schools
		WHERE id = $1
	`

	var s School
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.EducationSystemID, &s.Name, &s.IsActive, &s.IsDemo,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get school by id: %w", err)
	}
	return &s, nil
}

// GetActiveSchoolByUser returns the school associated with a user's active membership.
// Returns the school with the most privileged role if the user has multiple memberships.
func (r *SqlcRepository) GetActiveSchoolByUser(ctx context.Context, userID string) (*School, error) {
	const query = `
		SELECT sch.id, sch.tenant_id, sch.education_system_id, sch.name, sch.is_active, sch.is_demo
		FROM memberships m
		JOIN schools sch ON sch.id = m.school_id
		WHERE m.user_id = $1 AND m.is_active = true AND sch.is_active = true
		ORDER BY
			CASE m.role
				WHEN 'SYSTEM_ADMIN' THEN 1
				WHEN 'SCHOOL_ADMIN' THEN 2
				WHEN 'TEACHER' THEN 3
				WHEN 'SUPPORT_STAFF' THEN 4
			END
		LIMIT 1
	`

	var s School
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.ID, &s.TenantID, &s.EducationSystemID, &s.Name, &s.IsActive, &s.IsDemo,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active school by user: %w", err)
	}
	return &s, nil
}

// ActivateSchoolMembership sets is_active=false on all memberships for the user,
// then sets is_active=true on the target school membership.
func (r *SqlcRepository) ActivateSchoolMembership(ctx context.Context, userID, schoolID, tenantID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Deactivate all memberships for this user in this tenant
	_, err = tx.Exec(ctx,
		`UPDATE memberships SET is_active = false
		 WHERE user_id = $1 AND tenant_id = $2 AND school_id != $3`,
		userID, tenantID, schoolID,
	)
	if err != nil {
		return fmt.Errorf("deactivate memberships: %w", err)
	}

	// Activate the target membership (creates one if it doesn't exist)
	result, err := tx.Exec(ctx,
		`UPDATE memberships SET is_active = true
		 WHERE user_id = $1 AND school_id = $2 AND tenant_id = $3`,
		userID, schoolID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("activate membership: %w", err)
	}

	if result.RowsAffected() == 0 {
		// No membership row yet — this can happen if the school was created
		// by another admin and the user hasn't been assigned yet.
		// Default to TEACHER role.
		_, err = tx.Exec(ctx,
			`INSERT INTO memberships (user_id, school_id, tenant_id, role, is_active)
			 VALUES ($1, $2, $3, 'TEACHER'::user_role, true)
			 ON CONFLICT (user_id, school_id) DO UPDATE SET is_active = true`,
			userID, schoolID, tenantID,
		)
		if err != nil {
			return fmt.Errorf("insert new membership: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// Create inserts a new school and returns it.
func (r *SqlcRepository) Create(ctx context.Context, tenantID, name, educationSystemID string) (*School, error) {
	const query = `
		INSERT INTO schools (tenant_id, education_system_id, name)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, education_system_id, name, is_active, is_demo
	`

	var s School
	err := r.pool.QueryRow(ctx, query, tenantID, educationSystemID, name).Scan(
		&s.ID, &s.TenantID, &s.EducationSystemID, &s.Name, &s.IsActive, &s.IsDemo,
	)
	if err != nil {
		return nil, fmt.Errorf("insert school: %w", err)
	}
	return &s, nil
}

// UpdateName updates a school's name. Returns the updated school.
func (r *SqlcRepository) UpdateName(ctx context.Context, id, name string) (*School, error) {
	const query = `
		UPDATE schools SET name = $1
		WHERE id = $2 AND is_active = true
		RETURNING id, tenant_id, education_system_id, name, is_active, is_demo
	`

	var s School
	err := r.pool.QueryRow(ctx, query, name, id).Scan(
		&s.ID, &s.TenantID, &s.EducationSystemID, &s.Name, &s.IsActive, &s.IsDemo,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("update school name: %w", err)
	}
	return &s, nil
}

// Delete soft-deletes a school by setting is_active = false.
func (r *SqlcRepository) Delete(ctx context.Context, id string) error {
	const query = `UPDATE schools SET is_active = false WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft-delete school: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("school not found")
	}
	return nil
}

// CreateSchoolAndMembership creates a school and membership in a single transaction.
func (r *SqlcRepository) CreateSchoolAndMembership(
	ctx context.Context,
	tenantID, name, educationSystemID, userID, role string,
) (*School, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Insert school
	var s School
	err = tx.QueryRow(ctx,
		`INSERT INTO schools (tenant_id, education_system_id, name)
		 VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, education_system_id, name, is_active, is_demo`,
		tenantID, educationSystemID, name,
	).Scan(&s.ID, &s.TenantID, &s.EducationSystemID, &s.Name, &s.IsActive, &s.IsDemo)
	if err != nil {
		return nil, fmt.Errorf("insert school in tx: %w", err)
	}

	// Insert membership
	_, err = tx.Exec(ctx,
		`INSERT INTO memberships (tenant_id, role, user_id, school_id)
		 VALUES ($1, $2::user_role, $3, $4)
		 ON CONFLICT (user_id, school_id) DO UPDATE SET
			role = EXCLUDED.role,
			is_active = true,
			tenant_id = EXCLUDED.tenant_id`,
		tenantID, role, userID, s.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert membership in tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &s, nil
}
