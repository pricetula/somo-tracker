package teachers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles teacher database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ListBySchool returns paginated teachers for a given school.
// When includeInactive is true, both active and inactive memberships are returned.
// Search targets u.full_name, u.email, and u.tsc_number.
func (r *PgRepository) ListBySchool(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
	// Build the WHERE clause for active/inactive filter
	activeFilter := "TRUE"
	if !includeInactive {
		activeFilter = "m.is_active = true"
	}

	searchClause := ""
	var searchArgs []interface{}
	if search != "" {
		pattern := "%" + search + "%"
		searchClause = ` AND (u.full_name ILIKE $3 OR u.email ILIKE $4 OR u.tsc_number::text ILIKE $5)`
		searchArgs = []interface{}{pattern, pattern, pattern}
	}

	// Count total
	countArgs := []interface{}{tenantID, schoolID}
	countArgs = append(countArgs, searchArgs...)
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = 'TEACHER'
		  AND %s%s
	`, activeFilter, searchClause)

	var total int
	if err := database.FromContext(ctx, r.pool).QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("teachers.Repository.Count: %w", err)
	}

	// Fetch data with teacher-specific fields
	dataArgs := []interface{}{tenantID, schoolID}
	dataArgs = append(dataArgs, searchArgs...)
	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.email, u.full_name,
		       u.tsc_number, u.knec_panel_assessor_id,
		       cct.teacher_role,
		       m.is_active, m.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN LATERAL (
			SELECT teacher_role::text
			FROM cbc_class_teachers
			WHERE user_id = u.id
			  AND tenant_id = $1
			LIMIT 1
		) cct ON TRUE
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = 'TEACHER'
		  AND %s%s
	`, activeFilter, searchClause)

	argIdx := 3 + len(searchArgs)
	dataQuery += fmt.Sprintf(` ORDER BY u.full_name LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("teachers.Repository.ListBySchool: %w", err)
	}
	defer rows.Close()

	var teachers []Teacher
	for rows.Next() {
		var t Teacher
		if err := rows.Scan(
			&t.ID, &t.Email, &t.FullName,
			&t.TSCNumber, &t.KNECPanelAssessor,
			&t.TeacherRole,
			&t.IsActive, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("teachers.Repository.Scan: %w", err)
		}
		teachers = append(teachers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("teachers.Repository.Rows: %w", err)
	}

	if teachers == nil {
		teachers = []Teacher{}
	}

	return teachers, total, nil
}

// ToggleActive sets the is_active flag on a teacher's membership.
func (r *PgRepository) ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
	const query = `
		UPDATE memberships
		SET is_active = $1
		WHERE tenant_id = $2 AND school_id = $3 AND user_id = $4 AND role::text = 'TEACHER'
	`

	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, isActive, tenantID, schoolID, userID)
	if err != nil {
		return fmt.Errorf("teachers.Repository.ToggleActive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("teachers.Repository.ToggleActive: %w", ErrNotFound)
	}

	return nil
}

// Delete hard-deletes a teacher's membership and user record.
func (r *PgRepository) Delete(ctx context.Context, tenantID, schoolID, userID string) error {
	const query = `
		DELETE FROM memberships
		WHERE tenant_id = $1 AND school_id = $2 AND user_id = $3 AND role::text = 'TEACHER'
	`

	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, tenantID, schoolID, userID)
	if err != nil {
		return fmt.Errorf("teachers.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("teachers.Repository.Delete: %w", ErrNotFound)
	}

	return nil
}

// GetByID returns a single teacher by user ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, userID, tenantID, schoolID string) (*Teacher, error) {
	const query = `
		SELECT u.id, u.email, u.full_name,
		       u.tsc_number, u.knec_panel_assessor_id,
		       cct.teacher_role,
		       m.is_active, m.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN LATERAL (
			SELECT teacher_role::text
			FROM cbc_class_teachers
			WHERE user_id = u.id
			  AND tenant_id = $2
			LIMIT 1
		) cct ON TRUE
		WHERE m.tenant_id = $2 AND m.school_id = $3 AND m.user_id = $1 AND m.role::text = 'TEACHER'
	`

	var t Teacher
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, userID, tenantID, schoolID).Scan(
		&t.ID, &t.Email, &t.FullName,
		&t.TSCNumber, &t.KNECPanelAssessor,
		&t.TeacherRole,
		&t.IsActive, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("teachers.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("teachers.Repository.GetByID: %w", err)
	}
	return &t, nil
}

// Update applies partial updates to a teacher's user record.
func (r *PgRepository) Update(ctx context.Context, userID, tenantID, schoolID string, payload UpdateTeacherPayload) error {
	// Build dynamic SET clause from non-nil fields
	var sets []string
	args := []interface{}{userID}
	argIdx := 2

	if payload.FullName != nil {
		sets = append(sets, fmt.Sprintf("full_name = $%d", argIdx))
		args = append(args, *payload.FullName)
		argIdx++
	}
	if payload.TSCNumber != nil {
		sets = append(sets, fmt.Sprintf("tsc_number = $%d", argIdx))
		args = append(args, *payload.TSCNumber)
		argIdx++
	}
	if payload.KNECPanelAssessor != nil {
		sets = append(sets, fmt.Sprintf("knec_panel_assessor_id = $%d", argIdx))
		args = append(args, *payload.KNECPanelAssessor)
	}

	if len(sets) == 0 {
		return fmt.Errorf("teachers.Repository.Update: %w", ErrInvalidInput)
	}

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $1
	`, joinWithComma(sets))

	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("teachers.Repository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("teachers.Repository.Update: %w", ErrNotFound)
	}

	return nil
}

// joinWithComma joins a slice of strings with ", " separator.
func joinWithComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// ListTeacherClasses returns all classes assigned to a teacher in a given term.
func (r *PgRepository) ListTeacherClasses(ctx context.Context, tenantID, schoolID, userID, termID string) ([]TeacherClassItem, error) {
	query := `
		SELECT DISTINCT
			c.id AS class_id,
			c.grade_level || ' ' || COALESCE(s.name, '') AS class_name,
			c.grade_level,
			COALESCE(s.name, '') AS stream_name,
			at.id AS term_id,
			at.name AS term_name,
			COALESCE(ts.learning_area_id, '') AS learning_area_id,
			COALESCE(la.name, '') AS learning_area_name
		FROM cbc_class_teachers ct
		JOIN cbc_classes c ON c.id = ct.class_id AND c.tenant_id = $1
		LEFT JOIN cbc_streams s ON s.id = c.stream_id
		JOIN academic_terms at ON at.id = $4 AND at.tenant_id = $1 AND at.school_id = $2
		LEFT JOIN timetable_allocations ts ON ts.class_id = c.id AND ts.teacher_id = $3
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE ct.teacher_id = $3
		  AND c.school_id = $2
		  AND c.academic_year_id = at.academic_year_id
		ORDER BY class_name
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, userID, termID)
	if err != nil {
		return nil, fmt.Errorf("teachers.Repository.ListTeacherClasses: %w", err)
	}
	defer rows.Close()

	var items []TeacherClassItem
	for rows.Next() {
		var item TeacherClassItem
		if err := rows.Scan(
			&item.ClassID, &item.ClassName, &item.GradeLevel, &item.StreamName,
			&item.TermID, &item.TermName, &item.LearningAreaID, &item.LearningAreaName,
		); err != nil {
			return nil, fmt.Errorf("teachers.Repository.ListTeacherClasses: scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("teachers.Repository.ListTeacherClasses: rows: %w", err)
	}
	return items, nil
}

// GetTeacherTimetable returns the teacher's timetable slots for a given day of week.
func (r *PgRepository) GetTeacherTimetable(ctx context.Context, tenantID, schoolID, userID string, dayOfWeek int) ([]TeacherTimetableAllocation, error) {
	query := `
		SELECT
			ts.id AS timetable_allocation_id,
			tstr.period_name,
			tstr.start_time::text,
			tstr.end_time::text,
			c.id AS class_id,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			c.grade_level,
			COALESCE(st.name, '') AS stream_name,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name,
			ts.room_identifier
		FROM timetable_allocations ts
		JOIN timetable_blocks tstr ON tstr.id = ts.block_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = $1
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE ts.teacher_id = $3
		  AND ts.school_id = $2
		  AND tstr.day_of_week = $4
		  AND tstr.is_break = false
		ORDER BY tstr.start_time
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, userID, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("teachers.Repository.GetTeacherTimetable: %w", err)
	}
	defer rows.Close()

	var items []TeacherTimetableAllocation
	for rows.Next() {
		var item TeacherTimetableAllocation
		if err := rows.Scan(
			&item.TimetableAllocationID, &item.PeriodName, &item.StartTime, &item.EndTime,
			&item.ClassID, &item.ClassName, &item.GradeLevel, &item.StreamName,
			&item.LearningAreaID, &item.LearningAreaName, &item.RoomIdentifier,
		); err != nil {
			return nil, fmt.Errorf("teachers.Repository.GetTeacherTimetable: scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("teachers.Repository.GetTeacherTimetable: rows: %w", err)
	}
	return items, nil
}

// ListTeacherLessonTimeline returns flat weekly lesson items for the teacher's
// timeline view, paginated by week offset.
func (r *PgRepository) ListTeacherLessonTimeline(ctx context.Context, tenantID, schoolID, userID, weekStart string, limit int) ([]TeacherLessonTimelineItem, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `
		SELECT
			ts.id::text,
			COALESCE(la.name, '') AS subject_name,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			tstr.period_name,
			to_char(($4::date + (tstr.day_of_week - 1) * INTERVAL '1 day') + tstr.start_time, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS start_time,
			to_char(($4::date + (tstr.day_of_week - 1) * INTERVAL '1 day') + tstr.end_time, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS end_time,
			ts.room_identifier
		FROM timetable_allocations ts
		JOIN timetable_blocks tstr ON tstr.id = ts.block_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = $1
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE ts.teacher_id = $3
		  AND ts.school_id = $2
		  AND tstr.is_break = false
		ORDER BY (($4::date + (tstr.day_of_week - 1) * INTERVAL '1 day') + tstr.start_time) ASC
		LIMIT $5
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, userID, weekStart, limit)
	if err != nil {
		return nil, "", fmt.Errorf("teachers.Repository.ListTeacherLessonTimeline: %w", err)
	}
	defer rows.Close()

	items := make([]TeacherLessonTimelineItem, 0)
	for rows.Next() {
		var item TeacherLessonTimelineItem
		if err := rows.Scan(
			&item.ID,
			&item.SubjectName,
			&item.ClassName,
			&item.PeriodName,
			&item.StartTime,
			&item.EndTime,
			&item.Room,
		); err != nil {
			return nil, "", fmt.Errorf("teachers.Repository.ListTeacherLessonTimeline: scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("teachers.Repository.ListTeacherLessonTimeline: rows: %w", err)
	}

	nextCursor := ""
	if len(items) == limit {
		nextCursor = "1" // backend signals "another week is available" — frontend continues paginating by date
	}
	return items, nextCursor, nil
}

// GetActiveSchoolID returns the active school ID for a user in a tenant.
func (r *PgRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	const query = `
		SELECT school_id FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		ORDER BY
			CASE role
				WHEN 'SCHOOL_ADMIN'::user_role THEN 1
				WHEN 'TEACHER'::user_role THEN 2
				WHEN 'NURSE'::user_role THEN 3
				WHEN 'FINANCE'::user_role THEN 4
			END
		LIMIT 1
	`

	var schoolID string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID, userID).Scan(&schoolID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("teachers.Repository.GetActiveSchoolID: %w", ErrNotFound)
		}
		return "", fmt.Errorf("teachers.Repository.GetActiveSchoolID: %w", err)
	}
	return schoolID, nil
}
