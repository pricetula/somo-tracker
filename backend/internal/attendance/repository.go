package attendance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/timetable"
)

// PgRepository handles attendance database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// GetOrCreatePeriod returns an existing attendance period or creates a new one.
// Returns the period, a boolean indicating if it was newly created, and any error.
func (r *PgRepository) GetOrCreatePeriod(ctx context.Context, input OpenPeriodInput) (*AttendancePeriod, bool, error) {
	// Try to find an existing period first
	const findSQL = `
		SELECT id, tenant_id, school_id, academic_term_id, class_id,
		       cbc_learning_area_id, date_recorded::text, recorded_by,
		       authorized_by_role, created_at
		FROM cbc_attendance_periods
		WHERE tenant_id = $1
		  AND class_id = $2
		  AND date_recorded = $3
		  AND cbc_learning_area_id = $4
	`

	var period AttendancePeriod
	var role *timetable.TeacherRole
	err := r.pool.QueryRow(ctx, findSQL,
		input.TenantID, input.ClassID, input.Date, input.LearningAreaID,
	).Scan(
		&period.ID, &period.TenantID, &period.SchoolID, &period.AcademicTermID,
		&period.ClassID, &period.LearningAreaID, &period.DateRecorded,
		&period.RecordedBy, &role, &period.CreatedAt,
	)
	if err == nil {
		period.AuthorizedByRole = role
		return &period, false, nil
	}
	if err != pgx.ErrNoRows {
		return nil, false, fmt.Errorf("attendance.Repository.GetOrCreatePeriod: find: %w", err)
	}

	// Create new period
	const insertSQL = `
		INSERT INTO cbc_attendance_periods
			(tenant_id, school_id, academic_term_id, class_id,
			 cbc_learning_area_id, date_recorded, recorded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, school_id, academic_term_id, class_id,
		          cbc_learning_area_id, date_recorded::text, recorded_by,
		          authorized_by_role, created_at
	`

	err = r.pool.QueryRow(ctx, insertSQL,
		input.TenantID, input.SchoolID, input.AcademicTermID,
		input.ClassID, input.LearningAreaID, input.Date, input.RecordedBy,
	).Scan(
		&period.ID, &period.TenantID, &period.SchoolID, &period.AcademicTermID,
		&period.ClassID, &period.LearningAreaID, &period.DateRecorded,
		&period.RecordedBy, &role, &period.CreatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("attendance.Repository.GetOrCreatePeriod: insert: %w", err)
	}
	period.AuthorizedByRole = role

	return &period, true, nil
}

// UpdatePeriodAuthorizedBy sets the authorized_by_role on an attendance period.
func (r *PgRepository) UpdatePeriodAuthorizedBy(ctx context.Context, periodID, tenantID string, role timetable.TeacherRole) error {
	const updateSQL = `
		UPDATE cbc_attendance_periods
		SET authorized_by_role = $1
		WHERE id = $2 AND tenant_id = $3
	`
	_, err := r.pool.Exec(ctx, updateSQL, role, periodID, tenantID)
	if err != nil {
		return fmt.Errorf("attendance.Repository.UpdatePeriodAuthorizedBy: %w", err)
	}
	return nil
}

// UpsertAttendanceLogs batch upserts attendance logs for a period.
// Uses ON CONFLICT to update status in place without duplicates.
func (r *PgRepository) UpsertAttendanceLogs(ctx context.Context, tenantID, periodID, recordedBy string, logs []AttendanceLogInput) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("attendance.Repository.UpsertAttendanceLogs: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const upsertSQL = `
		INSERT INTO cbc_attendance_logs
			(tenant_id, cbc_attendance_period_id, student_id, status, remarks, recorded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cbc_attendance_period_id, student_id)
		DO UPDATE SET
			status = EXCLUDED.status,
			remarks = EXCLUDED.remarks,
			recorded_by = EXCLUDED.recorded_by
	`

	for _, log := range logs {
		_, err = tx.Exec(ctx, upsertSQL,
			tenantID, periodID, log.StudentID, log.Status, log.Remarks, recordedBy,
		)
		if err != nil {
			return fmt.Errorf("attendance.Repository.UpsertAttendanceLogs: exec: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("attendance.Repository.UpsertAttendanceLogs: commit tx: %w", err)
	}

	return nil
}

// GetPeriodLogs returns an attendance period with all its logs.
func (r *PgRepository) GetPeriodLogs(ctx context.Context, tenantID, periodID string) (*AttendancePeriod, []AttendanceLog, error) {
	// Fetch the period
	const periodSQL = `
		SELECT id, tenant_id, school_id, academic_term_id, class_id,
		       cbc_learning_area_id, date_recorded::text, recorded_by,
		       authorized_by_role, created_at
		FROM cbc_attendance_periods
		WHERE id = $1 AND tenant_id = $2
	`

	var period AttendancePeriod
	var role *timetable.TeacherRole
	err := r.pool.QueryRow(ctx, periodSQL, periodID, tenantID).Scan(
		&period.ID, &period.TenantID, &period.SchoolID, &period.AcademicTermID,
		&period.ClassID, &period.LearningAreaID, &period.DateRecorded,
		&period.RecordedBy, &role, &period.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("attendance.Repository.GetPeriodLogs: %w", ErrNotFound)
		}
		return nil, nil, fmt.Errorf("attendance.Repository.GetPeriodLogs: period: %w", err)
	}
	period.AuthorizedByRole = role

	// Fetch logs
	const logsSQL = `
		SELECT id, tenant_id, cbc_attendance_period_id, student_id,
		       status, remarks, recorded_by
		FROM cbc_attendance_logs
		WHERE tenant_id = $1 AND cbc_attendance_period_id = $2
		ORDER BY student_id
	`

	rows, err := r.pool.Query(ctx, logsSQL, tenantID, periodID)
	if err != nil {
		return nil, nil, fmt.Errorf("attendance.Repository.GetPeriodLogs: logs: %w", err)
	}
	defer rows.Close()

	var logs []AttendanceLog
	for rows.Next() {
		var l AttendanceLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.PeriodID, &l.StudentID,
			&l.Status, &l.Remarks, &l.RecordedBy,
		); err != nil {
			return nil, nil, fmt.Errorf("attendance.Repository.GetPeriodLogs: scan: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("attendance.Repository.GetPeriodLogs: rows: %w", err)
	}

	if logs == nil {
		logs = []AttendanceLog{}
	}

	return &period, logs, nil
}

// IsAuthorizedRecorder checks if a user is authorized to record attendance
// for a given class/learning area/term combination.
// This checks:
//   - SUBJECT_TEACHER assigned to the learning area in the class
//   - PRIMARY_CLASS_TEACHER on the class
//   - SUBSTITUTE_TEACHER on the class
//   - SCHOOL_ADMIN via memberships
func (r *PgRepository) IsAuthorizedRecorder(ctx context.Context, tenantID, userID, classID, learningAreaID, termID string) (*AuthorizedRecorderResult, error) {
	const query = `
		WITH user_roles AS (
			-- Check cbc_class_teachers for TEACHER roles
			SELECT teacher_role FROM cbc_class_teachers
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND class_id = $3
			  AND (
			      -- Subject teacher must match the learning area
			      (teacher_role = 'SUBJECT_TEACHER' AND learning_area_id = $4::uuid)
			      -- Primary and substitute are class-wide
			      OR teacher_role IN ('PRIMARY_CLASS_TEACHER', 'SUBSTITUTE_TEACHER')
			  )
			UNION ALL
			-- Check memberships for SCHOOL_ADMIN
			SELECT 'SCHOOL_ADMIN'::teacher_role FROM memberships
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND role = 'SCHOOL_ADMIN'
			  AND is_active = true
		)
		SELECT teacher_role FROM user_roles LIMIT 1
	`

	var role timetable.TeacherRole
	err := r.pool.QueryRow(ctx, query, tenantID, userID, classID, learningAreaID).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &AuthorizedRecorderResult{Authorized: false}, nil
		}
		return nil, fmt.Errorf("attendance.Repository.IsAuthorizedRecorder: %w", err)
	}

	return &AuthorizedRecorderResult{
		Authorized: true,
		Role:       &role,
	}, nil
}
