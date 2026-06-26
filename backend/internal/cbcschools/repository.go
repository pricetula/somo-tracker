package cbcschools

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles school database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// Create inserts a new cbc_school and returns its ID.
func (r *PgRepository) Create(ctx context.Context, tenantID string, name string) (string, error) {
	const query = `
		INSERT INTO cbc_schools (tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, '', '', 'Public')
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("cbcschools.Repository.Create: %w", err)
	}
	return id, nil
}

// ListByTenantID retrieves all schools for a tenant with member counts
// and whether each school is the user's currently active school.
func (r *PgRepository) ListByTenantID(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
	const query = `
		SELECT
			cs.id, cs.tenant_id, cs.name, cs.knec_school_code,
			cs.county, cs.sub_county, cs.ward,
			cs.school_type::text, cs.is_active, cs.created_at, cs.updated_at,
			COALESCE(smc.admins, 0) AS admins,
			COALESCE(smc.teachers, 0) AS teachers,
			COALESCE(smc.nurses, 0) AS nurses,
			COALESCE(smc.finance, 0) AS finance,
			COALESCE(smc.parents, 0) AS parents,
			COALESCE(smc.students, 0) AS students,
			CASE WHEN mas.school_id IS NOT NULL THEN true ELSE false END AS is_member_active_school
		FROM cbc_schools cs
		LEFT JOIN school_member_counts smc ON smc.school_id = cs.id
		LEFT JOIN member_active_school mas ON mas.school_id = cs.id AND mas.user_id = $2
		WHERE cs.tenant_id = $1
		ORDER BY cs.name ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("cbcschools.Repository.ListByTenantID: %w", err)
	}
	defer rows.Close()

	var schools []SchoolWithMemberCount
	for rows.Next() {
		var s SchoolWithMemberCount
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.KnecSchoolCode,
			&s.County, &s.SubCounty, &s.Ward,
			&s.SchoolType, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
			&s.Admins, &s.Teachers, &s.Nurses, &s.Finance, &s.Parents, &s.Students,
			&s.IsMemberActiveSchool,
		); err != nil {
			return nil, fmt.Errorf("cbcschools.Repository.ListByTenantID: scan: %w", err)
		}
		schools = append(schools, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbcschools.Repository.ListByTenantID: rows: %w", err)
	}

	if schools == nil {
		schools = []SchoolWithMemberCount{}
	}

	return schools, nil
}

// Update modifies school fields. Only non-nil fields are applied.
func (r *PgRepository) Update(ctx context.Context, school SchoolUpdateFields) error {
	// Build dynamic SET clause
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if school.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *school.Name)
		argIdx++
	}
	if school.County != nil {
		setClauses = append(setClauses, fmt.Sprintf("county = $%d", argIdx))
		args = append(args, *school.County)
		argIdx++
	}
	if school.SubCounty != nil {
		setClauses = append(setClauses, fmt.Sprintf("sub_county = $%d", argIdx))
		args = append(args, *school.SubCounty)
		argIdx++
	}
	if school.Ward != nil {
		setClauses = append(setClauses, fmt.Sprintf("ward = $%d", argIdx))
		args = append(args, *school.Ward)
		argIdx++
	}
	if school.KnecSchoolCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("knec_school_code = $%d", argIdx))
		args = append(args, *school.KnecSchoolCode)
		argIdx++
	}
	if school.NemisCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("nemis_institution_code = $%d", argIdx))
		args = append(args, *school.NemisCode)
		argIdx++
	}
	if school.SchoolType != nil {
		setClauses = append(setClauses, fmt.Sprintf("school_type = $%d::cbc_school_type", argIdx))
		args = append(args, *school.SchoolType)
		argIdx++
	}
	if school.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *school.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("cbcschools.Repository.Update: %w", ErrInvalidInput)
	}

	args = append(args, school.ID)
	query := fmt.Sprintf(`
		UPDATE cbc_schools
		SET %s
		WHERE id = $%d
	`, joinClauses(setClauses, ", "), argIdx)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("cbcschools.Repository.Update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("cbcschools.Repository.Update: %w", ErrNotFound)
	}

	return nil
}

// Delete removes a school by ID.
func (r *PgRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM cbc_schools WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("cbcschools.Repository.Delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("cbcschools.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// joinClauses joins strings with a separator. Helper for dynamic SET clauses.
func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	result := clauses[0]
	for _, c := range clauses[1:] {
		result += sep + c
	}
	return result
}

// GetByID retrieves a school by its ID.
func (r *PgRepository) GetByID(ctx context.Context, id string) (*School, error) {
	const query = `
		SELECT id, tenant_id, name, created_at
		FROM cbc_schools
		WHERE id = $1
	`
	var s School
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.TenantID, &s.Name, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcschools.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcschools.Repository.GetByID: %w", err)
	}
	return &s, nil
}
