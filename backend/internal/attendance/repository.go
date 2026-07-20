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
	err := r.pool.QueryRow(ctx, `
		INSERT INTO cbc_attendance_sessions (tenant_id, school_id, timetable_slot_id, date, status, skip_reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, school_id, timetable_slot_id, date, status, skip_reason, created_at
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
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, school_id, timetable_slot_id, date, status, skip_reason, created_at, updated_at
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
			s.id, s.tenant_id, s.school_id, s.timetable_slot_id, s.date, s.status, s.skip_reason,
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
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
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
			s.id, s.tenant_id, s.school_id, s.timetable_slot_id, s.date, s.status, s.skip_reason,
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

	rows, err := r.pool.Query(ctx, query, args...)
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
		RETURNING id, tenant_id, school_id, timetable_slot_id, date, status, skip_reason, created_at, updated_at
	`, setClause, argIdx, argIdx+1)

	args = append(args, id, tenantID)

	var s AttendanceSession
	err := r.pool.QueryRow(ctx, query, args...).Scan(
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
			s.id, s.tenant_id, s.school_id, s.timetable_slot_id, s.date, COALESCE(s.status, 'SUBMITTED') AS status, s.skip_reason,
			COALESCE(s.created_at, NOW()) AS created_at, COALESCE(s.updated_at, NOW()) AS updated_at,
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

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, classID, date)
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
	tx, err := r.pool.Begin(ctx)
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
	err := r.pool.QueryRow(ctx, query, args...).Scan(
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
	err := r.pool.QueryRow(ctx, `
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
		  AND ar.timetable_slot_id = $3 AND ar.date = $4
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

	rows, err := r.pool.Query(ctx, query, args...)
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
	rows, err := r.pool.Query(ctx, query, args...)
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

// ── Summaries ─────────────────────────────────────────────────────────────

func (r *pgRepository) GetStudentTermSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error) {
	query := `
		SELECT
			ats.id, ats.tenant_id, ats.school_id, ats.student_id, ats.academic_term_id,
			ats.learning_area_id, COALESCE(la.name, '') AS learning_area_name,
			ats.periods_total, ats.periods_present, ats.periods_absent,
			ats.periods_late, ats.periods_excused, ats.attendance_percentage,
			ats.last_refreshed_at, ats.updated_at
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
			ats.learning_area_id, COALESCE(la.name, '') AS learning_area_name,
			ats.periods_total, ats.periods_present, ats.periods_absent,
			ats.periods_late, ats.periods_excused, ats.attendance_percentage,
			ats.last_refreshed_at, ats.updated_at
		FROM attendance_term_summaries ats
		JOIN cbc_student_enrollments enr ON enr.student_id = ats.student_id
			AND enr.academic_term_id = ats.academic_term_id
			AND enr.school_id = ats.school_id
		LEFT JOIN cbc_learning_areas la ON la.id = ats.learning_area_id
		WHERE ats.tenant_id = $1 AND ats.school_id = $2
		  AND enr.class_id = $3 AND ats.academic_term_id = $4
		ORDER BY ats.student_id, la.name
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetClassTermSummary: %w", err)
	}
	defer rows.Close()

	var results []AttendanceTermSummary
	for rows.Next() {
		var s AttendanceTermSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.AcademicTermID,
			&s.LearningAreaID, &s.LearningAreaName,
			&s.PeriodsTotal, &s.PeriodsPresent, &s.PeriodsAbsent,
			&s.PeriodsLate, &s.PeriodsExcused, &s.AttendancePercentage,
			&s.LastRefreshedAt, &s.UpdatedAt,
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
	// This uses a single UPSERT query that aggregates raw attendance_records
	// per student × term × learning area and materialises the result.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO attendance_term_summaries (
			tenant_id, school_id, student_id, academic_term_id, learning_area_id,
			periods_total, periods_present, periods_absent, periods_late, periods_excused,
			attendance_percentage, last_refreshed_at
		)
		SELECT
			$1 AS tenant_id,
			ar.school_id,
			ar.student_id,
			ar.academic_term_id,
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
		WHERE ar.tenant_id = $1 AND ar.school_id = $2 AND ar.academic_term_id = $3
		GROUP BY ar.school_id, ar.student_id, ar.academic_term_id, ts.learning_area_id
		ON CONFLICT (student_id, academic_term_id, learning_area_id)
		DO UPDATE SET
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
func (r *pgRepository) scanSummaries(ctx context.Context, query string, args ...interface{}) ([]AttendanceTermSummary, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.scanSummaries: %w", err)
	}
	defer rows.Close()

	var results []AttendanceTermSummary
	for rows.Next() {
		var s AttendanceTermSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.AcademicTermID,
			&s.LearningAreaID, &s.LearningAreaName,
			&s.PeriodsTotal, &s.PeriodsPresent, &s.PeriodsAbsent,
			&s.PeriodsLate, &s.PeriodsExcused, &s.AttendancePercentage,
			&s.LastRefreshedAt, &s.UpdatedAt,
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

// Compile-time check
var _ Repository = (*pgRepository)(nil)
