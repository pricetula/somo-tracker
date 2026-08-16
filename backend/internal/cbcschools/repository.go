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
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID, name).Scan(&id)
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
		LEFT JOIN school_member_counts smc ON smc.school_id = cs.id AND smc.tenant_id = cs.tenant_id
		LEFT JOIN member_active_school mas ON mas.school_id = cs.id AND mas.user_id = $2
		WHERE cs.tenant_id = $1
		ORDER BY cs.name ASC
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, userID)
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

	result, err := database.FromContext(ctx, r.pool).Exec(ctx, query, args...)
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
	result, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id)
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
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id).Scan(&s.ID, &s.TenantID, &s.Name, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcschools.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcschools.Repository.GetByID: %w", err)
	}
	return &s, nil
}

// GetByTenantAndName retrieves a school within a tenant by its name.
// Returns ErrNotFound when the tenant has no school with that name.
// Used by the auth registration flow to reuse an existing school instead of
// creating a duplicate row when a second user registers for an existing tenant.
func (r *PgRepository) GetByTenantAndName(ctx context.Context, tenantID, name string) (*School, error) {
	const query = `
		SELECT id, tenant_id, name, created_at
		FROM cbc_schools
		WHERE tenant_id = $1 AND name = $2
		ORDER BY created_at ASC
		LIMIT 1
	`
	var s School
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID, name).Scan(&s.ID, &s.TenantID, &s.Name, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcschools.Repository.GetByTenantAndName: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcschools.Repository.GetByTenantAndName: %w", err)
	}
	return &s, nil
}

// OnboardingStatus returns the onboarding status for the given tenant.
func (r *PgRepository) OnboardingStatus(ctx context.Context, tenantID string) (*OnboardingStatus, error) {
	const query = `
		WITH tenant_context AS (
			SELECT $1::uuid AS current_tenant_id
		),

		-- 1. Check if class streams exist
		streams_check AS (
			SELECT 
				EXISTS (
					SELECT 1 
					FROM class_streams cs 
					JOIN tenant_context tc ON cs.tenant_id = tc.current_tenant_id
				) AS has_streams
		),

		-- 2. Verify all existing class streams have at least one timetable slot assigned
		timetable_check AS (
			SELECT 
				CASE 
					-- If no class streams exist, calendar setup cannot be complete
					WHEN NOT (SELECT has_streams FROM streams_check) THEN false
					-- Check if any class_stream exists WITHOUT an associated timetable entry
					ELSE NOT EXISTS (
						SELECT 1 
						FROM class_streams cs
						JOIN tenant_context tc ON cs.tenant_id = tc.current_tenant_id
						WHERE NOT EXISTS (
							SELECT 1 
							FROM timetable_slots ts 
							WHERE ts.class_stream_id = cs.id 
							  AND ts.tenant_id = cs.tenant_id
						)
					)
				END AS calendar_configured
		),

		-- 3. Check curriculum initialization (learning areas, strands, and sub-strands)
		curriculum_check AS (
			SELECT (
				EXISTS (SELECT 1 FROM learning_areas la JOIN tenant_context tc ON la.tenant_id = tc.current_tenant_id)
				AND EXISTS (SELECT 1 FROM cbc_strands cs JOIN tenant_context tc ON cs.tenant_id = tc.current_tenant_id)
				AND EXISTS (SELECT 1 FROM cbc_sub_strands css JOIN tenant_context tc ON css.tenant_id = tc.current_tenant_id)
			) AS curriculum_initialized
		),

		-- 4. Check staff invitation (At least one teacher)
		staff_check AS (
			SELECT EXISTS (
				SELECT 1 
				FROM users u 
				JOIN tenant_context tc ON u.tenant_id = tc.current_tenant_id
				WHERE u.role = 'TEACHER'
			) AS staff_invited
		),

		-- 5. Check student enrollment
		student_check AS (
			SELECT EXISTS (
				SELECT 1 
				FROM cbc_students s 
				JOIN tenant_context tc ON s.tenant_id = tc.current_tenant_id
			) AS students_enrolled
		)

		SELECT 
			tc.current_tenant_id AS tenant_id,
			sc.has_streams AS class_streams_created,
			tc_check.calendar_configured AS academic_calendar_configured,
			cc.curriculum_initialized,
			st.staff_invited,
			st_check.students_enrolled,
			(
				tc_check.calendar_configured 
				AND cc.curriculum_initialized 
				AND sc.has_streams 
				AND st.staff_invited 
				AND st_check.students_enrolled
			) AS is_onboarding_complete
		FROM tenant_context tc
		CROSS JOIN streams_check sc
		CROSS JOIN timetable_check tc_check
		CROSS JOIN curriculum_check cc
		CROSS JOIN staff_check st
		CROSS JOIN student_check st_check;
	`

	var status OnboardingStatus
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID).Scan(
		&status.TenantID,
		&status.ClassStreamsCreated,
		&status.AcademicCalendarConfigured,
		&status.CurriculumInitialized,
		&status.StaffInvited,
		&status.StudentsEnrolled,
		&status.IsOnboardingComplete,
	)
	if err != nil {
		return nil, fmt.Errorf("cbcschools.Repository.OnboardingStatus: %w", err)
	}
	return &status, nil
}
