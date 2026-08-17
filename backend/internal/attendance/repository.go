package attendance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed attendance repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

// ── Sessions ──────────────────────────────────────────────────────────────

func (r *pgRepository) CreateSession(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error) {
	var s AttendanceSession
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO cbc_attendance_sessions (tenant_id, school_id, timetable_slot_id, date, status, skip_reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, school_id, timetable_slot_id, date::text, status, skip_reason, created_at
	`, tenantID, schoolID, payload.TimetableSlotID, payload.Date, payload.Status, payload.SkipReason).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.TimetableSlotID, &s.Date, &s.Status, &s.SkipReason, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.CreateSession: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetSessionByID(ctx context.Context, id, tenantID string) (*AttendanceSession, error) {
	var s AttendanceSession
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT id, tenant_id, school_id, timetable_slot_id, date::text, status, skip_reason, created_at, updated_at
		FROM cbc_attendance_sessions
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.TimetableSlotID, &s.Date, &s.Status, &s.SkipReason, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetSessionByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetSessionByID: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetEnrichedSessionByID(ctx context.Context, id, tenantID string) (*SessionWithEnrichedData, error) {
	query := `
		SELECT
			s.id, s.tenant_id, s.school_id, s.timetable_slot_id, s.date::text, s.status, s.skip_reason,
			s.created_at, s.updated_at,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			COALESCE(st.name, '') AS stream_name,
			c.grade_level,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name,
			u.full_name AS teacher_name
		FROM cbc_attendance_sessions s
		JOIN cbc_timetable_slots ts ON ts.id = s.timetable_slot_id
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = s.tenant_id
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		LEFT JOIN users u ON u.id = ts.teacher_id AND u.tenant_id = s.tenant_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`
	var res SessionWithEnrichedData
	var streamName, learningAreaName, teacherName string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID).Scan(
		&res.ID, &res.TenantID, &res.SchoolID, &res.TimetableSlotID, &res.Date, &res.Status, &res.SkipReason,
		&res.CreatedAt, &res.UpdatedAt,
		&res.ClassName, &streamName, &res.GradeLevel,
		&res.PeriodName, &res.DayOfWeek, &res.StartTime, &res.EndTime,
		&res.LearningAreaID, &learningAreaName, &teacherName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetEnrichedSessionByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetEnrichedSessionByID: %w", err)
	}
	res.StreamName = streamName
	res.LearningArea = &learningAreaName
	if teacherName != "" {
		res.TeacherName = &teacherName
	}
	return &res, nil
}

func (r *pgRepository) ListSessions(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error) {
	query := `
		SELECT
			s.id, s.tenant_id, s.school_id, s.timetable_slot_id, s.date::text, s.status, s.skip_reason,
			s.created_at, s.updated_at,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			COALESCE(st.name, '') AS stream_name,
			c.grade_level,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name,
			u.full_name AS teacher_name
		FROM cbc_attendance_sessions s
		JOIN cbc_timetable_slots ts ON ts.id = s.timetable_slot_id
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = s.tenant_id
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		LEFT JOIN users u ON u.id = ts.teacher_id AND u.tenant_id = s.tenant_id
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND s.tenant_id = $%d", argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.SchoolID != "" {
		query += fmt.Sprintf(" AND s.school_id = $%d", argIdx)
		args = append(args, filter.SchoolID)
		argIdx++
	}
	if filter.TimetableSlotID != "" {
		query += fmt.Sprintf(" AND s.timetable_slot_id = $%d", argIdx)
		args = append(args, filter.TimetableSlotID)
		argIdx++
	}
	if filter.Date != "" {
		query += fmt.Sprintf(" AND s.date = $%d", argIdx)
		args = append(args, filter.Date)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND s.status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.ClassID != "" {
		query += fmt.Sprintf(" AND ts.class_id = $%d", argIdx)
		args = append(args, filter.ClassID)
	}

	query += ` ORDER BY s.date DESC, tstr.day_of_week, tstr.start_time`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListSessions: %w", err)
	}
	defer rows.Close()

	var results []SessionWithEnrichedData
	for rows.Next() {
		var res SessionWithEnrichedData
		var streamName, learningAreaName, teacherName string
		if err := rows.Scan(
			&res.ID, &res.TenantID, &res.SchoolID, &res.TimetableSlotID, &res.Date, &res.Status, &res.SkipReason,
			&res.CreatedAt, &res.UpdatedAt,
			&res.ClassName, &streamName, &res.GradeLevel,
			&res.PeriodName, &res.DayOfWeek, &res.StartTime, &res.EndTime,
			&res.LearningAreaID, &learningAreaName, &teacherName,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListSessions: scan: %w", err)
		}
		res.StreamName = streamName
		res.LearningArea = &learningAreaName
		if teacherName != "" {
			res.TeacherName = &teacherName
		}
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListSessions: rows: %w", err)
	}
	return results, nil
}

func (r *pgRepository) UpdateSession(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error) {
	setClause := ""
	args := []interface{}{}
	argIdx := 1

	if payload.Status != nil {
		setClause += fmt.Sprintf("status = $%d, ", argIdx)
		args = append(args, *payload.Status)
		argIdx++
	}
	if payload.SkipReason != nil {
		setClause += fmt.Sprintf("skip_reason = $%d, ", argIdx)
		args = append(args, *payload.SkipReason)
		argIdx++
	}

	if setClause == "" {
		return nil, fmt.Errorf("attendance.Repository.UpdateSession: no fields to update: %w", ErrInvalidInput)
	}

	setClause = setClause[:len(setClause)-2]

	query := fmt.Sprintf(`
		UPDATE cbc_attendance_sessions
		SET %s
		WHERE id = $%d AND tenant_id = $%d
		RETURNING id, tenant_id, school_id, timetable_slot_id, date::text, status, skip_reason, created_at, updated_at
	`, setClause, argIdx, argIdx+1)

	args = append(args, id, tenantID)

	var s AttendanceSession
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.TimetableSlotID, &s.Date, &s.Status, &s.SkipReason, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.UpdateSession: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.UpdateSession: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetSessionsForClassDate(ctx context.Context, tenantID, schoolID, classID, date string) ([]SessionWithEnrichedData, error) {
	// Return all non-break timetable slots for this class on this day of week,
	// along with any existing session record.
	query := `
		SELECT
			COALESCE(s.id::text, gen_random_uuid()::text) AS id,
			COALESCE(s.tenant_id, $1::uuid) AS tenant_id,
			COALESCE(s.school_id, $2::uuid) AS school_id,
			COALESCE(s.timetable_slot_id, ts.id) AS timetable_slot_id,
			COALESCE(s.date::text, $4::text) AS date,
			COALESCE(s.status, 'SUBMITTED') AS status,
			s.skip_reason,
			COALESCE(s.created_at, NOW()) AS created_at,
			COALESCE(s.updated_at, NOW()) AS updated_at,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			COALESCE(st.name, '') AS stream_name,
			c.grade_level,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name,
			u.full_name AS teacher_name
		FROM cbc_timetable_slots ts
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = $1
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		LEFT JOIN users u ON u.id = ts.teacher_id AND u.tenant_id = $1
		LEFT JOIN cbc_attendance_sessions s
			ON s.timetable_slot_id = ts.id AND s.date = $4 AND s.tenant_id = $1
		WHERE ts.class_id = $3
		  AND c.school_id = $2
		  AND tstr.is_break = false
		  AND tstr.day_of_week = (
		      SELECT EXTRACT(DOW FROM $4::DATE)::INT
		  )
		ORDER BY tstr.start_time
	`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, classID, date)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetSessionsForClassDate: %w", err)
	}
	defer rows.Close()

	var results []SessionWithEnrichedData
	for rows.Next() {
		var res SessionWithEnrichedData
		var streamName, learningAreaName, teacherName string
		if err := rows.Scan(
			&res.ID, &res.TenantID, &res.SchoolID, &res.TimetableSlotID, &res.Date, &res.Status, &res.SkipReason,
			&res.CreatedAt, &res.UpdatedAt,
			&res.ClassName, &streamName, &res.GradeLevel,
			&res.PeriodName, &res.DayOfWeek, &res.StartTime, &res.EndTime,
			&res.LearningAreaID, &learningAreaName, &teacherName,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetSessionsForClassDate: scan: %w", err)
		}
		res.StreamName = streamName
		res.LearningArea = &learningAreaName
		if teacherName != "" {
			res.TeacherName = &teacherName
		}
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetSessionsForClassDate: rows: %w", err)
	}
	return results, nil
}

// ── Records ───────────────────────────────────────────────────────────────

func (r *pgRepository) BatchMark(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy string, termID string) (*BatchMarkResult, error) {
	result := &BatchMarkResult{}

	// Use a transaction so the batch is atomic.
	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.BatchMark: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, rec := range payload.Records {
		tag, err := tx.Exec(ctx, `
			INSERT INTO attendance_records
				(tenant_id, school_id, student_id, timetable_slot_id, academic_term_id,
				 date, status, marked_by, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (student_id, timetable_slot_id, date)
			DO UPDATE SET
				status = EXCLUDED.status,
				note = COALESCE(EXCLUDED.note, attendance_records.note),
				marked_by = EXCLUDED.marked_by,
				marked_at = NOW()
		`, tenantID, schoolID, rec.StudentID, payload.TimetableSlotID, termID,
			payload.Date, string(rec.Status), markedBy, rec.Note,
		)
		if err != nil {
			return nil, fmt.Errorf("attendance.Repository.BatchMark: upsert: %w", err)
		}
		rowsAffected := tag.RowsAffected()
		if rowsAffected == 1 {
			// INSERT (new row)
			result.Created++
		} else {
			// UPDATE (existing row)
			result.Updated++
		}
	}

	// Upsert a session record so the timeline recognises this slot+date as completed.
	// The unique constraint is on (school_id, timetable_slot_id, date).
	_, err = tx.Exec(ctx, `
		INSERT INTO cbc_attendance_sessions
			(tenant_id, school_id, timetable_slot_id, date, status)
		VALUES ($1, $2, $3, $4, 'SUBMITTED')
		ON CONFLICT (school_id, timetable_slot_id, date)
		DO UPDATE SET
			status = 'SUBMITTED'
	`, tenantID, schoolID, payload.TimetableSlotID, payload.Date)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.BatchMark: upsert session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("attendance.Repository.BatchMark: commit tx: %w", err)
	}

	return result, nil
}

func (r *pgRepository) UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error) {
	setClause := ""
	args := []interface{}{}
	argIdx := 1

	if payload.Status != nil {
		setClause += fmt.Sprintf("status = $%d, ", argIdx)
		args = append(args, string(*payload.Status))
		argIdx++
	}
	if payload.Note != nil {
		setClause += fmt.Sprintf("note = $%d, ", argIdx)
		args = append(args, *payload.Note)
		argIdx++
	}

	if setClause == "" {
		return nil, fmt.Errorf("attendance.Repository.UpdateRecord: no fields to update: %w", ErrInvalidInput)
	}

	setClause = setClause[:len(setClause)-2]

	query := fmt.Sprintf(`
		UPDATE attendance_records
		SET %s
		WHERE id = $%d AND tenant_id = $%d
		RETURNING id, tenant_id, school_id, student_id, timetable_slot_id, academic_term_id,
		          date, status, marked_by, marked_at, note, created_at, updated_at
	`, setClause, argIdx, argIdx+1)

	args = append(args, id, tenantID)

	var rec AttendanceRecord
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, args...).Scan(
		&rec.ID, &rec.TenantID, &rec.SchoolID, &rec.StudentID, &rec.TimetableSlotID, &rec.AcademicTermID,
		&rec.Date, &rec.Status, &rec.MarkedBy, &rec.MarkedBy, &rec.Note, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.UpdateRecord: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.UpdateRecord: %w", err)
	}
	return &rec, nil
}

func (r *pgRepository) GetRecordByID(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
	var rec AttendanceRecord
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT id, tenant_id, school_id, student_id, timetable_slot_id, academic_term_id,
		       date, status, marked_by, marked_at, note, created_at, updated_at
		FROM attendance_records
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&rec.ID, &rec.TenantID, &rec.SchoolID, &rec.StudentID, &rec.TimetableSlotID, &rec.AcademicTermID,
		&rec.Date, &rec.Status, &rec.MarkedBy, &rec.MarkedBy, &rec.Note, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetRecordByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetRecordByID: %w", err)
	}
	return &rec, nil
}

func (r *pgRepository) ListRecordsBySlotDate(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) ([]RecordWithEnrichedData, error) {
	query := `
		SELECT
			COALESCE(ar.id::text, gen_random_uuid()::text) AS id,
			$1::text AS tenant_id,
			$2::text AS school_id,
			s.id::text AS student_id,
			ts.id::text AS timetable_slot_id,
			COALESCE(ar.academic_term_id::text, enr.academic_term_id::text) AS academic_term_id,
			$4::text AS date,
			COALESCE(ar.status, 'PRESENT') AS status,
			COALESCE(ar.marked_by::text, '') AS marked_by,
			COALESCE(ar.marked_at::text, '') AS marked_at,
			ar.note,
			COALESCE(ar.created_at, NOW()) AS created_at,
			COALESCE(ar.updated_at, NOW()) AS updated_at,
			s.full_name AS student_full_name,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			c.grade_level,
			COALESCE(st.name, '') AS stream_name,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name
		FROM cbc_timetable_slots ts
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = $1
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		JOIN cbc_student_enrollments enr
			ON enr.class_id = c.id
			AND enr.tenant_id = $1
			AND enr.school_id = $2
		JOIN cbc_students s ON s.id = enr.student_id AND s.tenant_id = $1
		LEFT JOIN attendance_records ar
			ON ar.student_id = s.id
			AND ar.timetable_slot_id = ts.id
			AND ar.date = $4::date
			AND ar.tenant_id = $1
		WHERE ts.id = $3
		  AND ts.tenant_id = $1
		ORDER BY s.full_name
	`
	return r.scanEnrichedRecords(ctx, query, tenantID, schoolID, timetableSlotID, date)
}

func (r *pgRepository) ListRecordsByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]RecordWithEnrichedData, error) {
	query := `
		SELECT
			ar.id, ar.tenant_id, ar.school_id, ar.student_id, ar.timetable_slot_id,
			ar.academic_term_id, ar.date, ar.status, ar.marked_by, ar.marked_at, ar.note,
			ar.created_at, ar.updated_at,
			s.full_name AS student_full_name,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			c.grade_level,
			COALESCE(st.name, '') AS stream_name,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name
		FROM attendance_records ar
		JOIN cbc_students s ON s.id = ar.student_id AND s.tenant_id = ar.tenant_id
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = ar.tenant_id
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE ar.tenant_id = $1 AND ar.school_id = $2
		  AND ar.student_id = $3 AND ar.academic_term_id = $4
		ORDER BY ar.date DESC, tstr.start_time
	`
	return r.scanEnrichedRecords(ctx, query, tenantID, schoolID, studentID, termID)
}

func (r *pgRepository) ListRecordsByClassDate(ctx context.Context, tenantID, schoolID, classID, date string) ([]RecordWithEnrichedData, error) {
	query := `
		SELECT
			ar.id, ar.tenant_id, ar.school_id, ar.student_id, ar.timetable_slot_id,
			ar.academic_term_id, ar.date, ar.status, ar.marked_by, ar.marked_at, ar.note,
			ar.created_at, ar.updated_at,
			s.full_name AS student_full_name,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			c.grade_level,
			COALESCE(st.name, '') AS stream_name,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name
		FROM attendance_records ar
		JOIN cbc_students s ON s.id = ar.student_id AND s.tenant_id = ar.tenant_id
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = ar.tenant_id
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE ar.tenant_id = $1 AND ar.school_id = $2
		  AND c.id = $3 AND ar.date = $4
		ORDER BY s.full_name, tstr.start_time
	`
	return r.scanEnrichedRecords(ctx, query, tenantID, schoolID, classID, date)
}

func (r *pgRepository) ListRecords(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error) {
	query := `
		SELECT
			ar.id, ar.tenant_id, ar.school_id, ar.student_id, ar.timetable_slot_id,
			ar.academic_term_id, ar.date, ar.status, ar.marked_by, ar.marked_at, ar.note,
			ar.created_at, ar.updated_at,
			s.full_name AS student_full_name,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			c.grade_level,
			COALESCE(st.name, '') AS stream_name,
			tstr.period_name,
			tstr.day_of_week,
			tstr.start_time::text,
			tstr.end_time::text,
			ts.learning_area_id,
			COALESCE(la.name, '') AS learning_area_name
		FROM attendance_records ar
		JOIN cbc_students s ON s.id = ar.student_id AND s.tenant_id = ar.tenant_id
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN timetable_structures tstr ON tstr.id = ts.structure_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = ar.tenant_id
		LEFT JOIN cbc_streams st ON st.id = c.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND ar.tenant_id = $%d", argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.SchoolID != "" {
		query += fmt.Sprintf(" AND ar.school_id = $%d", argIdx)
		args = append(args, filter.SchoolID)
		argIdx++
	}
	if filter.StudentID != "" {
		query += fmt.Sprintf(" AND ar.student_id = $%d", argIdx)
		args = append(args, filter.StudentID)
		argIdx++
	}
	if filter.TimetableSlotID != "" {
		query += fmt.Sprintf(" AND ar.timetable_slot_id = $%d", argIdx)
		args = append(args, filter.TimetableSlotID)
		argIdx++
	}
	if filter.Date != "" {
		query += fmt.Sprintf(" AND ar.date = $%d", argIdx)
		args = append(args, filter.Date)
		argIdx++
	}
	if filter.AcademicTermID != "" {
		query += fmt.Sprintf(" AND ar.academic_term_id = $%d", argIdx)
		args = append(args, filter.AcademicTermID)
		argIdx++
	}
	if filter.ClassID != "" {
		query += fmt.Sprintf(" AND c.id = $%d", argIdx)
		args = append(args, filter.ClassID)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND ar.status = $%d", argIdx)
		args = append(args, filter.Status)
	}

	query += ` ORDER BY ar.date DESC, s.full_name, tstr.start_time`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListRecords: %w", err)
	}
	defer rows.Close()

	var results []RecordWithEnrichedData
	for rows.Next() {
		var rec RecordWithEnrichedData
		var streamName, learningAreaName string
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.SchoolID, &rec.StudentID, &rec.TimetableSlotID,
			&rec.AcademicTermID, &rec.Date, &rec.Status, &rec.MarkedBy, &rec.MarkedBy, &rec.Note,
			&rec.CreatedAt, &rec.UpdatedAt,
			&rec.StudentFullName, &rec.ClassName, &rec.GradeLevel, &streamName,
			&rec.PeriodName, &rec.DayOfWeek, &rec.StartTime, &rec.EndTime,
			&rec.LearningAreaID, &learningAreaName,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListRecords: scan: %w", err)
		}
		rec.StreamName = streamName
		rec.LearningAreaName = &learningAreaName
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListRecords: rows: %w", err)
	}
	return results, nil
}

// scanEnrichedRecords is a shared helper for the concrete list-by-query methods.
func (r *pgRepository) scanEnrichedRecords(ctx context.Context, query string, args ...interface{}) ([]RecordWithEnrichedData, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.scanEnrichedRecords: %w", err)
	}
	defer rows.Close()

	var results []RecordWithEnrichedData
	for rows.Next() {
		var rec RecordWithEnrichedData
		var streamName, learningAreaName string
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.SchoolID, &rec.StudentID, &rec.TimetableSlotID,
			&rec.AcademicTermID, &rec.Date, &rec.Status, &rec.MarkedBy, &rec.MarkedBy, &rec.Note,
			&rec.CreatedAt, &rec.UpdatedAt,
			&rec.StudentFullName, &rec.ClassName, &rec.GradeLevel, &streamName,
			&rec.PeriodName, &rec.DayOfWeek, &rec.StartTime, &rec.EndTime,
			&rec.LearningAreaID, &learningAreaName,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.scanEnrichedRecords: scan: %w", err)
		}
		rec.StreamName = streamName
		rec.LearningAreaName = &learningAreaName
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.scanEnrichedRecords: rows: %w", err)
	}
	return results, nil
}

// ── Term Resolution ───────────────────────────────────────────────────────

func (r *pgRepository) GetTermIDByDate(ctx context.Context, tenantID, schoolID, date string) (string, error) {
	var id string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT at.id FROM academic_terms at
		JOIN academic_years ay ON ay.id = at.academic_year_id
		WHERE ay.tenant_id = $1
		  AND ay.school_id = $2
		  AND at.start_date <= $3::date
		  AND at.end_date >= $3::date
		LIMIT 1
	`, tenantID, schoolID, date).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("attendance.Repository.GetTermIDByDate: no term found for date %s: %w", date, ErrInvalidInput)
		}
		return "", fmt.Errorf("attendance.Repository.GetTermIDByDate: %w", err)
	}
	return id, nil
}

// ── Summaries ─────────────────────────────────────────────────────────────

func (r *pgRepository) GetStudentTermSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error) {
	query := `
		SELECT
			ats.id, ats.tenant_id, ats.school_id, ats.student_id, ats.academic_term_id,
			ats.academic_year_id,
			ats.learning_area_id, COALESCE(la.name, '') AS learning_area_name,
			ats.periods_total, ats.periods_present, ats.periods_absent,
			ats.periods_late, ats.periods_excused, ats.attendance_percentage,
			ats.last_refreshed_at, ats.created_at, ats.updated_at
		FROM attendance_term_summaries ats
		LEFT JOIN cbc_learning_areas la ON la.id = ats.learning_area_id
		WHERE ats.tenant_id = $1 AND ats.school_id = $2
		  AND ats.student_id = $3 AND ats.academic_term_id = $4
		ORDER BY la.name
	`
	return r.scanSummaries(ctx, query, tenantID, schoolID, studentID, termID)
}

func (r *pgRepository) GetClassTermSummary(ctx context.Context, tenantID, schoolID, classID, termID string) ([]AttendanceTermSummary, error) {
	query := `
		SELECT
			ats.id, ats.tenant_id, ats.school_id, ats.student_id, ats.academic_term_id,
			ats.academic_year_id,
			ats.learning_area_id, COALESCE(la.name, '') AS learning_area_name,
			ats.periods_total, ats.periods_present, ats.periods_absent,
			ats.periods_late, ats.periods_excused, ats.attendance_percentage,
			ats.last_refreshed_at, ats.created_at, ats.updated_at
		FROM attendance_term_summaries ats
		JOIN cbc_student_enrollments enr ON enr.student_id = ats.student_id
			AND enr.academic_term_id = ats.academic_term_id
			AND enr.school_id = ats.school_id
		LEFT JOIN cbc_learning_areas la ON la.id = ats.learning_area_id
		WHERE ats.tenant_id = $1 AND ats.school_id = $2
		  AND enr.class_id = $3 AND ats.academic_term_id = $4
		ORDER BY ats.student_id, la.name
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetClassTermSummary: %w", err)
	}
	defer rows.Close()

	var results []AttendanceTermSummary
	for rows.Next() {
		var s AttendanceTermSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.AcademicTermID,
			&s.AcademicYearID,
			&s.LearningAreaID, &s.LearningAreaName,
			&s.PeriodsTotal, &s.PeriodsPresent, &s.PeriodsAbsent,
			&s.PeriodsLate, &s.PeriodsExcused, &s.AttendancePercentage,
			&s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetClassTermSummary: scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetClassTermSummary: rows: %w", err)
	}
	return results, nil
}

func (r *pgRepository) RefreshSummaries(ctx context.Context, tenantID, schoolID, termID string) error {
	// Recompute all attendance term summaries for a given term.
	// Excludes attendance_records whose session is SKIPPED (cancelled lesson)
	// so cancelled lessons don't count against the denominator.
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, `
		INSERT INTO attendance_term_summaries (
			tenant_id, school_id, student_id, academic_term_id, academic_year_id,
			learning_area_id,
			periods_total, periods_present, periods_absent, periods_late, periods_excused,
			attendance_percentage, last_refreshed_at
		)
		SELECT
			$1 AS tenant_id,
			ar.school_id,
			ar.student_id,
			ar.academic_term_id,
			t.academic_year_id,
			ts.learning_area_id,
			COUNT(*)::INT AS periods_total,
			COUNT(*) FILTER (WHERE ar.status = 'PRESENT')::INT AS periods_present,
			COUNT(*) FILTER (WHERE ar.status = 'ABSENT')::INT AS periods_absent,
			COUNT(*) FILTER (WHERE ar.status = 'LATE')::INT AS periods_late,
			COUNT(*) FILTER (WHERE ar.status = 'EXCUSED')::INT AS periods_excused,
			ROUND(
				(COUNT(*) FILTER (WHERE ar.status = 'PRESENT') * 100.0 / NULLIF(COUNT(*), 0)),
				2
			) AS attendance_percentage,
			NOW() AS last_refreshed_at
		FROM attendance_records ar
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN academic_terms t ON t.id = ar.academic_term_id
		LEFT JOIN cbc_attendance_sessions s
			ON s.timetable_slot_id = ar.timetable_slot_id
			AND s.date = ar.date
			AND s.tenant_id = ar.tenant_id
		WHERE ar.tenant_id = $1 AND ar.school_id = $2 AND ar.academic_term_id = $3
		  AND (s.status IS NULL OR s.status != 'SKIPPED')
		GROUP BY ar.school_id, ar.student_id, ar.academic_term_id, t.academic_year_id, ts.learning_area_id
		ON CONFLICT (student_id, academic_term_id, learning_area_id)
		DO UPDATE SET
			academic_year_id     = EXCLUDED.academic_year_id,
			periods_total        = EXCLUDED.periods_total,
			periods_present      = EXCLUDED.periods_present,
			periods_absent       = EXCLUDED.periods_absent,
			periods_late         = EXCLUDED.periods_late,
			periods_excused      = EXCLUDED.periods_excused,
			attendance_percentage = EXCLUDED.attendance_percentage,
			last_refreshed_at    = EXCLUDED.last_refreshed_at
	`, tenantID, schoolID, termID)
	if err != nil {
		return fmt.Errorf("attendance.Repository.RefreshSummaries: %w", err)
	}
	return nil
}

// scanSummaries is a shared helper for scanning summary rows.
// Order must match SELECT: id, tenant_id, school_id, student_id, academic_term_id,
// academic_year_id, learning_area_id, learning_area_name, periods_total,
// periods_present, periods_absent, periods_late, periods_excused,
// attendance_percentage, last_refreshed_at, created_at, updated_at.
func (r *pgRepository) scanSummaries(ctx context.Context, query string, args ...interface{}) ([]AttendanceTermSummary, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.scanSummaries: %w", err)
	}
	defer rows.Close()

	var results []AttendanceTermSummary
	for rows.Next() {
		var s AttendanceTermSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.AcademicTermID,
			&s.AcademicYearID,
			&s.LearningAreaID, &s.LearningAreaName,
			&s.PeriodsTotal, &s.PeriodsPresent, &s.PeriodsAbsent,
			&s.PeriodsLate, &s.PeriodsExcused, &s.AttendancePercentage,
			&s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.scanSummaries: scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.scanSummaries: rows: %w", err)
	}
	return results, nil
}

// ── Class Daily Summaries ─────────────────────────────────────────────────

func (r *pgRepository) GetClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) (*ClassDailyAttendanceSummary, error) {
	var s ClassDailyAttendanceSummary
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT id, tenant_id, school_id, class_id, academic_term_id, date::TEXT,
		       total_enrolled, present_count, absent_count, late_count, excused_count,
		       daily_attendance_rate, last_refreshed_at, created_at, updated_at
		FROM class_daily_attendance_summaries
		WHERE tenant_id = $1 AND school_id = $2 AND class_id = $3 AND date = $4
	`, tenantID, schoolID, classID, date).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.AcademicTermID, &s.Date,
		&s.TotalEnrolled, &s.PresentCount, &s.AbsentCount, &s.LateCount, &s.ExcusedCount,
		&s.DailyAttendanceRate, &s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetClassDailySummary: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetClassDailySummary: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) RefreshClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) error {
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, `
		INSERT INTO class_daily_attendance_summaries (
			tenant_id, school_id, class_id, academic_term_id, date,
			total_enrolled, present_count, absent_count, late_count, excused_count,
			daily_attendance_rate, last_refreshed_at
		)
		SELECT
			$1 AS tenant_id,
			$2 AS school_id,
			c.id AS class_id,
			ar.academic_term_id,
			ar.date,
			COUNT(DISTINCT ar.student_id)::INT AS total_enrolled,
			COUNT(*) FILTER (WHERE ar.status = 'PRESENT')::INT AS present_count,
			COUNT(*) FILTER (WHERE ar.status = 'ABSENT')::INT AS absent_count,
			COUNT(*) FILTER (WHERE ar.status = 'LATE')::INT AS late_count,
			COUNT(*) FILTER (WHERE ar.status = 'EXCUSED')::INT AS excused_count,
			ROUND(
				(COUNT(*) FILTER (WHERE ar.status = 'PRESENT') * 100.0 / NULLIF(COUNT(*), 0)),
				2
			) AS daily_attendance_rate,
			NOW() AS last_refreshed_at
		FROM attendance_records ar
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = $1
		LEFT JOIN cbc_attendance_sessions s
			ON s.timetable_slot_id = ar.timetable_slot_id
			AND s.date = ar.date
			AND s.tenant_id = ar.tenant_id
		WHERE ar.tenant_id = $1
		  AND ar.school_id = $2
		  AND c.id = $3
		  AND ar.date = $4
		  AND (s.status IS NULL OR s.status != 'SKIPPED')
		GROUP BY c.id, ar.academic_term_id, ar.date
		ON CONFLICT (class_id, date)
		DO UPDATE SET
			total_enrolled        = EXCLUDED.total_enrolled,
			present_count         = EXCLUDED.present_count,
			absent_count          = EXCLUDED.absent_count,
			late_count            = EXCLUDED.late_count,
			excused_count         = EXCLUDED.excused_count,
			daily_attendance_rate = EXCLUDED.daily_attendance_rate,
			last_refreshed_at     = EXCLUDED.last_refreshed_at
	`, tenantID, schoolID, classID, date)
	if err != nil {
		return fmt.Errorf("attendance.Repository.RefreshClassDailySummary: %w", err)
	}
	return nil
}

func (r *pgRepository) ListClassDailySummaries(ctx context.Context, tenantID, schoolID, classID, startDate, endDate string) ([]ClassDailyAttendanceSummary, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, `
		SELECT id, tenant_id, school_id, class_id, academic_term_id, date::TEXT,
		       total_enrolled, present_count, absent_count, late_count, excused_count,
		       daily_attendance_rate, last_refreshed_at, created_at, updated_at
		FROM class_daily_attendance_summaries
		WHERE tenant_id = $1 AND school_id = $2 AND class_id = $3
		  AND date >= $4 AND date <= $5
		ORDER BY date ASC
	`, tenantID, schoolID, classID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassDailySummaries: %w", err)
	}
	defer rows.Close()

	var results []ClassDailyAttendanceSummary
	for rows.Next() {
		var s ClassDailyAttendanceSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.AcademicTermID, &s.Date,
			&s.TotalEnrolled, &s.PresentCount, &s.AbsentCount, &s.LateCount, &s.ExcusedCount,
			&s.DailyAttendanceRate, &s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListClassDailySummaries: scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassDailySummaries: rows: %w", err)
	}
	return results, nil
}

// ── Calendar Status ──────────────────────────────────────────────────────

func (r *pgRepository) ListCalendarStatus(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error) {
	query := `
		WITH dates AS (
			SELECT d::DATE AS dt
			FROM generate_series($3::DATE, $4::DATE, '1 day'::INTERVAL) d
		),
		expected AS (
			SELECT
				d.dt AS date,
				ts.id AS timetable_slot_id
			FROM dates d
			JOIN timetable_structures tstr
				ON tstr.school_id = $2 AND tstr.is_break = false
				AND tstr.day_of_week = EXTRACT(ISODOW FROM d.dt)::INT
			JOIN cbc_timetable_slots ts
				ON ts.structure_id = tstr.id AND ts.school_id = $2
			WHERE tstr.tenant_id = $1
			  AND ts.tenant_id = $1
		),
		handled AS (
			SELECT DISTINCT e.date, e.timetable_slot_id
			FROM expected e
			WHERE EXISTS (
				SELECT 1 FROM attendance_records ar
				WHERE ar.timetable_slot_id = e.timetable_slot_id
				  AND ar.date = e.date
				  AND ar.tenant_id = $1
			)
			OR EXISTS (
				SELECT 1 FROM cbc_attendance_sessions cas
				WHERE cas.timetable_slot_id = e.timetable_slot_id
				  AND cas.date = e.date
				  AND cas.tenant_id = $1
				  AND cas.status = 'SKIPPED'
			)
		)
		SELECT
			e.date::TEXT,
			COUNT(DISTINCT e.timetable_slot_id)::INT AS expected_count,
			COUNT(DISTINCT h.timetable_slot_id)::INT AS handled_count
		FROM expected e
		LEFT JOIN handled h ON h.date = e.date AND h.timetable_slot_id = e.timetable_slot_id
		GROUP BY e.date
		ORDER BY e.date ASC
	`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListCalendarStatus: %w", err)
	}
	defer rows.Close()

	var results []CalendarDayStatusRaw
	for rows.Next() {
		var s CalendarDayStatusRaw
		if err := rows.Scan(
			&s.Date, &s.ExpectedCount, &s.HandledCount,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListCalendarStatus: scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListCalendarStatus: rows: %w", err)
	}
	return results, nil
}

// ── Class Learning Area Term Summaries ─────────────────────────────────

func (r *pgRepository) GetClassLearningAreaTermSummary(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) (*ClassLearningAreaTermSummary, error) {
	var s ClassLearningAreaTermSummary
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT id, tenant_id, school_id, class_id, learning_area_id, academic_term_id, academic_year_id,
		       students_included, periods_total, periods_present, periods_absent, periods_late, periods_excused,
		       attendance_percentage, last_refreshed_at, created_at, updated_at
		FROM class_learning_area_term_summaries
		WHERE tenant_id = $1 AND school_id = $2 AND class_id = $3 AND learning_area_id = $4 AND academic_term_id = $5
	`, tenantID, schoolID, classID, learningAreaID, termID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.LearningAreaID, &s.AcademicTermID, &s.AcademicYearID,
		&s.StudentsIncluded, &s.PeriodsTotal, &s.PeriodsPresent, &s.PeriodsAbsent, &s.PeriodsLate, &s.PeriodsExcused,
		&s.AttendancePercentage, &s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetClassLearningAreaTermSummary: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetClassLearningAreaTermSummary: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) ListClassLearningAreaTermSummaries(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) ([]ClassLearningAreaTermSummary, error) {
	// Build dynamic WHERE clause based on optional filters.
	query := `
		SELECT id, tenant_id, school_id, class_id, learning_area_id, academic_term_id, academic_year_id,
		       students_included, periods_total, periods_present, periods_absent, periods_late, periods_excused,
		       attendance_percentage, last_refreshed_at, created_at, updated_at
		FROM class_learning_area_term_summaries
		WHERE tenant_id = $1 AND school_id = $2 AND academic_term_id = $3`
	args := []interface{}{tenantID, schoolID, termID}

	if classID != "" {
		query += " AND class_id = $4"
		args = append(args, classID)
		if learningAreaID != "" {
			query += fmt.Sprintf(" AND learning_area_id = $%d", len(args)+1)
			args = append(args, learningAreaID)
		}
	} else if learningAreaID != "" {
		query += fmt.Sprintf(" AND learning_area_id = $%d", len(args)+1)
		args = append(args, learningAreaID)
	}

	query += " ORDER BY class_id, learning_area_id"

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassLearningAreaTermSummaries: %w", err)
	}
	defer rows.Close()

	var results []ClassLearningAreaTermSummary
	for rows.Next() {
		var s ClassLearningAreaTermSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.LearningAreaID, &s.AcademicTermID, &s.AcademicYearID,
			&s.StudentsIncluded, &s.PeriodsTotal, &s.PeriodsPresent, &s.PeriodsAbsent, &s.PeriodsLate, &s.PeriodsExcused,
			&s.AttendancePercentage, &s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListClassLearningAreaTermSummaries: scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassLearningAreaTermSummaries: rows: %w", err)
	}
	return results, nil
}

// ── Class Term Attendance Summaries ───────────────────────────────────

func (r *pgRepository) GetClassTermAttendanceSummary(ctx context.Context, tenantID, schoolID, classID, termID string) (*ClassTermAttendanceSummary, error) {
	var s ClassTermAttendanceSummary
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT id, tenant_id, school_id, class_id, academic_term_id, academic_year_id,
		       days_in_term, total_enrolled_avg, present_count, absent_count, late_count, excused_count,
		       term_attendance_rate, last_refreshed_at, created_at, updated_at
		FROM class_term_attendance_summaries
		WHERE tenant_id = $1 AND school_id = $2 AND class_id = $3 AND academic_term_id = $4
	`, tenantID, schoolID, classID, termID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.AcademicTermID, &s.AcademicYearID,
		&s.DaysInTerm, &s.TotalEnrolledAvg, &s.PresentCount, &s.AbsentCount, &s.LateCount, &s.ExcusedCount,
		&s.TermAttendanceRate, &s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetClassTermAttendanceSummary: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetClassTermAttendanceSummary: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) ListClassTermAttendanceSummaries(ctx context.Context, tenantID, schoolID, classID, termID string) ([]ClassTermAttendanceSummary, error) {
	var rows pgx.Rows
	var err error
	if classID == "" {
		rows, err = database.FromContext(ctx, r.pool).Query(ctx, `
			SELECT id, tenant_id, school_id, class_id, academic_term_id, academic_year_id,
			       days_in_term, total_enrolled_avg, present_count, absent_count, late_count, excused_count,
			       term_attendance_rate, last_refreshed_at, created_at, updated_at
			FROM class_term_attendance_summaries
			WHERE tenant_id = $1 AND school_id = $2 AND academic_term_id = $3
			ORDER BY class_id
		`, tenantID, schoolID, termID)
	} else {
		rows, err = database.FromContext(ctx, r.pool).Query(ctx, `
			SELECT id, tenant_id, school_id, class_id, academic_term_id, academic_year_id,
			       days_in_term, total_enrolled_avg, present_count, absent_count, late_count, excused_count,
			       term_attendance_rate, last_refreshed_at, created_at, updated_at
			FROM class_term_attendance_summaries
			WHERE tenant_id = $1 AND school_id = $2 AND class_id = $3 AND academic_term_id = $4
			ORDER BY class_id
		`, tenantID, schoolID, classID, termID)
	}
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassTermAttendanceSummaries: %w", err)
	}
	defer rows.Close()

	var results []ClassTermAttendanceSummary
	for rows.Next() {
		var s ClassTermAttendanceSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.AcademicTermID, &s.AcademicYearID,
			&s.DaysInTerm, &s.TotalEnrolledAvg, &s.PresentCount, &s.AbsentCount, &s.LateCount, &s.ExcusedCount,
			&s.TermAttendanceRate, &s.LastRefreshedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListClassTermAttendanceSummaries: scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassTermAttendanceSummaries: rows: %w", err)
	}
	return results, nil
}

// ListClassAttendanceBreakdowns returns per-class Present/Late/Absent counts
// for a school in a term, LEFT JOINing cbc_classes so every class appears even
// when its term summary has not been materialised yet (COALESCE to zero).
// Ordered by absent_count DESC NULLS LAST — high-absenteeism classes first.
func (r *pgRepository) ListClassAttendanceBreakdowns(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, `
		SELECT
			c.id AS class_id,
			c.grade_level || ' ' || COALESCE(st.name, '') AS class_name,
			COALESCE(ctas.total_enrolled_avg, 0) AS total_enrolled_avg,
			COALESCE(ctas.present_count, 0) AS present_count,
			COALESCE(ctas.late_count, 0) AS late_count,
			COALESCE(ctas.absent_count, 0) AS absent_count,
			COALESCE(ctas.excused_count, 0) AS excused_count,
			COALESCE(ctas.term_attendance_rate, 0.00) AS term_attendance_rate
		FROM cbc_classes c
		LEFT JOIN cbc_streams st ON st.tenant_id = c.tenant_id AND st.id = c.stream_id
		LEFT JOIN class_term_attendance_summaries ctas
			ON c.tenant_id = ctas.tenant_id
			AND c.id = ctas.class_id
			AND ctas.academic_term_id = $3
		WHERE c.tenant_id = $1
		  AND c.school_id = $2
		ORDER BY ctas.absent_count DESC NULLS LAST
	`, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassAttendanceBreakdowns: %w", err)
	}
	defer rows.Close()

	var results []ClassAttendanceBreakdownItem
	for rows.Next() {
		var item ClassAttendanceBreakdownItem
		if err := rows.Scan(
			&item.ClassID, &item.ClassName, &item.TotalEnrolledAvg,
			&item.PresentCount, &item.LateCount, &item.AbsentCount,
			&item.ExcusedCount, &item.TermAttendanceRate,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListClassAttendanceBreakdowns: scan: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListClassAttendanceBreakdowns: rows: %w", err)
	}
	return results, nil
}

// ListLearningAreaBreakdowns returns per-learning-area Present/Absent/Excused
// period counts for a school in a term, aggregated across all classes via
// cbc_learning_areas LEFT JOIN class_learning_area_term_summaries. Learning
// areas with no summaries in the term still surface with zeroed counts
// (LEFT JOIN), and the whole set is ordered by periods_absent DESC so the
// highest-absenteeism subjects — the truancy/disengagement hotspot watch —
// appear first in the School Administrator dashboard grouped bar chart.
func (r *pgRepository) ListLearningAreaBreakdowns(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, `
		SELECT
			l.id AS learning_area_id,
			l.name AS learning_area_name,
			COALESCE(SUM(lts.periods_total), 0)::INT AS periods_total,
			COALESCE(SUM(lts.periods_present), 0)::INT AS periods_present,
			COALESCE(SUM(lts.periods_absent), 0)::INT AS periods_absent,
			COALESCE(SUM(lts.periods_excused), 0)::INT AS periods_excused,
			CASE
				WHEN SUM(lts.periods_total) > 0
				THEN ROUND((SUM(lts.periods_present)::NUMERIC / SUM(lts.periods_total)) * 100, 2)
				ELSE 0.00
			END AS attendance_percentage
		FROM cbc_learning_areas l
		LEFT JOIN class_learning_area_term_summaries lts
			ON l.tenant_id = lts.tenant_id
			AND l.id = lts.learning_area_id
			AND lts.academic_term_id = $3
		WHERE l.tenant_id = $1
		  AND l.school_id = $2
		GROUP BY l.id, l.name
		ORDER BY periods_absent DESC NULLS LAST
	`, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListLearningAreaBreakdowns: %w", err)
	}
	defer rows.Close()

	var results []LearningAreaAttendanceBreakdownItem
	for rows.Next() {
		var item LearningAreaAttendanceBreakdownItem
		if err := rows.Scan(
			&item.LearningAreaID, &item.LearningAreaName,
			&item.PeriodsTotal, &item.PeriodsPresent, &item.PeriodsAbsent,
			&item.PeriodsExcused, &item.AttendancePercentage,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.ListLearningAreaBreakdowns: scan: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.ListLearningAreaBreakdowns: rows: %w", err)
	}
	return results, nil
}

// ── School Attendance KPIs ──────────────────────────────────────────────

func (r *pgRepository) GetSchoolAttendanceKPIs(ctx context.Context, tenantID, schoolID, date, termID string) (*SchoolAttendanceKPI, error) {
	// An empty termID (no active term covers the date) degrades the term-rate
	// CTE to zero rows → COALESCE → 0.00 instead of failing the whole request.
	var termParam interface{} = nil
	if termID != "" {
		termParam = termID
	}

	var kpi SchoolAttendanceKPI
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		WITH today_summary AS (
			SELECT
				COALESCE(SUM(present_count), 0)::INT AS total_present,
				COALESCE(SUM(present_count + absent_count + late_count + excused_count), 0)::INT AS total_marked_records,
				COALESCE(AVG(daily_attendance_rate), 0.00)::NUMERIC AS avg_daily_rate
			FROM class_daily_attendance_summaries
			WHERE tenant_id = $1
			  AND school_id = $2
			  AND date = $3::DATE
		),
		term_summary AS (
			SELECT
				COALESCE(AVG(term_attendance_rate), 0.00)::NUMERIC AS active_term_attendance_rate
			FROM class_term_attendance_summaries
			WHERE tenant_id = $1
			  AND school_id = $2
			  AND academic_term_id = $4::UUID
		),
		unmarked_slots AS (
			-- Non-break timetable slots for the school on this weekday with no
			-- attendance session record yet (neither SUBMITTED nor SKIPPED).
			-- is_break / day_of_week live on timetable_structures, not on
			-- cbc_timetable_slots; both are per-school so the join is fully
			-- scoped by tenant + school.
			SELECT COUNT(*)::INT AS unmarked_count
			FROM cbc_timetable_slots ts
			JOIN timetable_structures tstr
				ON tstr.id = ts.structure_id
				AND tstr.tenant_id = ts.tenant_id
				AND tstr.school_id = ts.school_id
			WHERE ts.tenant_id = $1
			  AND ts.school_id = $2
			  AND tstr.is_break = FALSE
			  AND tstr.day_of_week = EXTRACT(ISODOW FROM $3::DATE)::INT
			  AND NOT EXISTS (
				  SELECT 1 FROM cbc_attendance_sessions cas
				  WHERE cas.tenant_id = ts.tenant_id
					AND cas.timetable_slot_id = ts.id
					AND cas.date = $3::DATE
			  )
		),
		skipped_sessions AS (
			SELECT COUNT(*)::INT AS skipped_count
			FROM cbc_attendance_sessions cas
			WHERE cas.tenant_id = $1
			  AND cas.school_id = $2
			  AND cas.date = $3::DATE
			  AND cas.status = 'SKIPPED'
		)
		SELECT
			tsum.avg_daily_rate::FLOAT8 AS todays_attendance_rate,
			tsum.total_present,
			tsum.total_marked_records,
			trm.active_term_attendance_rate::FLOAT8 AS active_term_attendance_rate,
			ums.unmarked_count AS unmarked_slots_today,
			sks.skipped_count AS skipped_sessions_today
		FROM today_summary tsum
		CROSS JOIN term_summary trm
		CROSS JOIN unmarked_slots ums
		CROSS JOIN skipped_sessions sks
	`, tenantID, schoolID, date, termParam).Scan(
		&kpi.TodaysAttendanceRate,
		&kpi.TotalPresent,
		&kpi.TotalMarkedRecords,
		&kpi.ActiveTermAttendanceRate,
		&kpi.UnmarkedSlotsToday,
		&kpi.SkippedSessionsToday,
	)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetSchoolAttendanceKPIs: %w", err)
	}
	return &kpi, nil
}

// Compile-time check
var _ Repository = (*pgRepository)(nil)
