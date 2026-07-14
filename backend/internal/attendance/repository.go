package attendance

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository defines the contract for attendance persistence.
type Repository interface {
	// GetRosterForSlot returns the list of students enrolled in the class
	// associated with a timetable slot, along with any existing attendance marks.
	GetRosterForSlot(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error)

	// BulkUpsert inserts or updates attendance records for a given slot and date.
	BulkUpsert(ctx context.Context, tenantID, schoolID string, payload BulkAttendancePayload, markedBy string) error

	// GetStudentHistory returns a student's raw attendance records filtered by parameters.
	GetStudentHistory(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error)

	// UpdateRecord updates a single attendance record (admin correction).
	UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error

	// GetAdminDashboard returns completion status for all classes on a given date.
	GetAdminDashboard(ctx context.Context, tenantID, schoolID, date string) (*AdminDashboardResponse, error)

	// GetChildAttendanceSummary returns a summarised attendance view for a parent's child.
	GetChildAttendanceSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error)

	// ComputeTermSummaries recalculates attendance_term_summaries for all students
	// in a given school and term. Returns count of rows upserted.
	ComputeTermSummaries(ctx context.Context, tenantID, schoolID, termID string) (int, error)

	// GetRecordsBySlotDate returns all attendance records for a given slot + date.
	GetRecordsBySlotDate(ctx context.Context, timetableSlotID, date string) ([]AttendanceRecord, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed attendance repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

func (r *pgRepository) GetRosterForSlot(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error) {
	query := `
		SELECT
			ts.class_id,
			c.grade_level || ' ' || COALESCE(str.name, '') AS class_name,
			COALESCE(la.name, '') AS learning_area,
			ts.id AS slot_id
		FROM cbc_timetable_slots ts
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = ts.tenant_id
		LEFT JOIN cbc_streams str ON str.id = c.stream_id
		WHERE ts.id = $1 AND ts.tenant_id = $2
	`
	var (
		classID      string
		className    string
		learningArea string
		slotID       string
	)
	err := r.pool.QueryRow(ctx, query, timetableSlotID, tenantID).Scan(&classID, &className, &learningArea, &slotID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: %w", err)
	}

	// Get enrolled students for this class
	studentQuery := `
		SELECT
			s.id AS student_id,
			s.full_name,
			COALESCE(s.admission_number, '') AS admission_number
		FROM cbc_student_enrollments e
		JOIN cbc_students s ON s.id = e.student_id AND s.tenant_id = e.tenant_id
		WHERE e.class_id = $1
		  AND e.tenant_id = $2
		  AND e.status = 'ACTIVE'
		ORDER BY s.full_name
	`
	rows, err := r.pool.Query(ctx, studentQuery, classID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: %w", err)
	}
	defer rows.Close()

	var students []RosterStudent
	var studentIDs []string
	studentMap := make(map[string]*RosterStudent)

	for rows.Next() {
		var s RosterStudent
		if err := rows.Scan(&s.StudentID, &s.FullName, &s.AdmissionNumber); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: scan student: %w", err)
		}
		students = append(students, s)
		studentIDs = append(studentIDs, s.StudentID)
		studentMap[s.StudentID] = &students[len(students)-1]
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: rows iteration: %w", err)
	}

	// Fetch existing attendance records for this slot + date
	if len(studentIDs) > 0 {
		attQuery := `
			SELECT student_id, status
			FROM attendance_records
			WHERE timetable_slot_id = $1 AND date = $2 AND tenant_id = $3
		`
		attRows, err := r.pool.Query(ctx, attQuery, timetableSlotID, date, tenantID)
		if err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: fetch existing marks: %w", err)
		}
		defer attRows.Close()

		for attRows.Next() {
			var studentID string
			var status AttendanceStatus
			if err := attRows.Scan(&studentID, &status); err != nil {
				return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: scan mark: %w", err)
			}
			if s, ok := studentMap[studentID]; ok {
				s.CurrentStatus = &status
			}
		}
		if err := attRows.Err(); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetRosterForSlot: rows iteration: %w", err)
		}
	}

	return &SlotRosterResponse{
		TimetableSlotID: timetableSlotID,
		ClassName:       className,
		LearningArea:    learningArea,
		Date:            date,
		Students:        students,
	}, nil
}

func (r *pgRepository) BulkUpsert(ctx context.Context, tenantID, schoolID string, payload BulkAttendancePayload, markedBy string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("attendance.Repository.BulkUpsert: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Get academic_term_id from the timetable slot
	var termID string
	termQuery := `
		SELECT at.id
		FROM cbc_timetable_slots ts
		JOIN academic_terms at ON at.academic_year_id = ts.academic_year_id
			AND at.school_id = $1 AND at.tenant_id = $2
			AND at.is_current = true
		WHERE ts.id = $3 AND ts.tenant_id = $2
	`
	err = tx.QueryRow(ctx, termQuery, schoolID, tenantID, payload.TimetableSlotID).Scan(&termID)
	if err != nil {
		return fmt.Errorf("attendance.Repository.BulkUpsert: resolve term: %w", err)
	}

	now := time.Now()
	for _, entry := range payload.Entries {
		_, err = tx.Exec(ctx, `
			INSERT INTO attendance_records
				(tenant_id, school_id, student_id, timetable_slot_id, academic_term_id,
				 date, status, marked_by, marked_at, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (student_id, timetable_slot_id, date)
			DO UPDATE SET
				status     = EXCLUDED.status,
				marked_by  = EXCLUDED.marked_by,
				marked_at  = EXCLUDED.marked_at,
				note       = EXCLUDED.note
		`,
			tenantID, schoolID, entry.StudentID, payload.TimetableSlotID, termID,
			payload.Date, entry.Status, markedBy, now, entry.Note,
		)
		if err != nil {
			return fmt.Errorf("attendance.Repository.BulkUpsert: upsert entry %s: %w", entry.StudentID, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("attendance.Repository.BulkUpsert: commit: %w", err)
	}

	return nil
}

func (r *pgRepository) GetRecordsBySlotDate(ctx context.Context, timetableSlotID, date string) ([]AttendanceRecord, error) {
	query := `
		SELECT id, tenant_id, school_id, student_id, timetable_slot_id,
		       academic_term_id, date, status, marked_by, marked_at, note, created_at
		FROM attendance_records
		WHERE timetable_slot_id = $1 AND date = $2
		ORDER BY student_id
	`
	rows, err := r.pool.Query(ctx, query, timetableSlotID, date)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetRecordsBySlotDate: %w", err)
	}
	defer rows.Close()

	var records []AttendanceRecord
	for rows.Next() {
		var rec AttendanceRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.SchoolID, &rec.StudentID, &rec.TimetableSlotID,
			&rec.AcademicTermID, &rec.Date, &rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Note, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetRecordsBySlotDate: scan: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *pgRepository) GetStudentHistory(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error) {
	query := `
		SELECT ar.id, ar.tenant_id, ar.school_id, ar.student_id, ar.timetable_slot_id,
		       ar.academic_term_id, ar.date, ar.status, ar.marked_by, ar.marked_at, ar.note, ar.created_at
		FROM attendance_records ar
		WHERE ar.student_id = $1 AND ar.tenant_id = $2
	`
	args := []interface{}{studentID, tenantID}
	argIdx := 3

	if filter.TermID != "" {
		query += fmt.Sprintf(" AND ar.academic_term_id = $%d", argIdx)
		args = append(args, filter.TermID)
		argIdx++
	}
	if filter.StartDate != "" {
		query += fmt.Sprintf(" AND ar.date >= $%d", argIdx)
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != "" {
		query += fmt.Sprintf(" AND ar.date <= $%d", argIdx)
		args = append(args, filter.EndDate)
	}

	query += " ORDER BY ar.date DESC, ar.timetable_slot_id"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetStudentHistory: %w", err)
	}
	defer rows.Close()

	var records []AttendanceRecord
	for rows.Next() {
		var rec AttendanceRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.SchoolID, &rec.StudentID, &rec.TimetableSlotID,
			&rec.AcademicTermID, &rec.Date, &rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Note, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetStudentHistory: scan: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *pgRepository) UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE attendance_records
		SET status = $1, note = $2
		WHERE id = $3 AND tenant_id = $4
	`, payload.Status, payload.Note, id, tenantID)
	if err != nil {
		return fmt.Errorf("attendance.Repository.UpdateRecord: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("attendance.Repository.UpdateRecord: %w", ErrNotFound)
	}
	return nil
}

func (r *pgRepository) GetAdminDashboard(ctx context.Context, tenantID, schoolID, date string) (*AdminDashboardResponse, error) {
	// For the given date, find all non-break timetable slots for the school
	// and count how many have attendance records marked.
	query := `
		SELECT
			c.grade_level || ' ' || COALESCE(str.name, '') AS class_name,
			ts.id AS slot_id,
			ts_period.period_name,
			COUNT(DISTINCT e.student_id) AS total_students,
			COUNT(DISTINCT ar.student_id) FILTER (WHERE ar.id IS NOT NULL) AS marked_students
		FROM cbc_timetable_slots ts
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = ts.tenant_id
		LEFT JOIN cbc_streams str ON str.id = c.stream_id
		JOIN timetable_structures ts_period ON ts_period.id = ts.structure_id
			AND ts_period.tenant_id = ts.tenant_id
			AND ts_period.is_break = false
		JOIN cbc_student_enrollments e ON e.class_id = ts.class_id AND e.tenant_id = ts.tenant_id
			AND e.status = 'ACTIVE'
		LEFT JOIN attendance_records ar ON ar.timetable_slot_id = ts.id
			AND ar.date = $3 AND ar.student_id = e.student_id
		WHERE ts.tenant_id = $1 AND ts.school_id = $2
		GROUP BY c.grade_level || ' ' || COALESCE(str.name, ''), ts.id, ts_period.period_name
		ORDER BY ts_period.period_name
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, date)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetAdminDashboard: %w", err)
	}
	defer rows.Close()

	var classes []CompletionStatus
	for rows.Next() {
		var cs CompletionStatus
		if err := rows.Scan(&cs.ClassName, &cs.SlotID, &cs.PeriodName, &cs.TotalSlots, &cs.MarkedSlots); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetAdminDashboard: scan: %w", err)
		}
		cs.IsComplete = cs.MarkedSlots >= cs.TotalSlots
		classes = append(classes, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetAdminDashboard: rows iteration: %w", err)
	}

	return &AdminDashboardResponse{Date: date, Classes: classes}, nil
}

func (r *pgRepository) GetChildAttendanceSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error) {
	// Try the materialised summary first, fall back to computing on the fly.
	var summary AttendanceTermSummary
	summaryQuery := `
		SELECT periods_total, periods_present, periods_absent, periods_late,
		       periods_excused, attendance_percentage
		FROM attendance_term_summaries
		WHERE student_id = $1 AND academic_term_id = $2
			AND tenant_id = $3
		ORDER BY last_refreshed_at DESC
		LIMIT 1
	`
	err := r.pool.QueryRow(ctx, summaryQuery, studentID, termID, tenantID).Scan(
		&summary.PeriodsTotal, &summary.PeriodsPresent, &summary.PeriodsAbsent,
		&summary.PeriodsLate, &summary.PeriodsExcused, &summary.AttendancePercentage,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("attendance.Repository.GetChildAttendanceSummary: summary: %w", err)
	}

	// Fetch recent periods (last 30 days)
	recentQuery := `
		SELECT ar.date, COALESCE(la.name, '') AS subject, ar.status
		FROM attendance_records ar
		LEFT JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		LEFT JOIN cbc_learning_areas la ON la.id = ts.learning_area_id
		WHERE ar.student_id = $1
		  AND ar.tenant_id = $2
		  AND ar.date >= CURRENT_DATE - INTERVAL '30 days'
		ORDER BY ar.date DESC, ts.structure_id
		LIMIT 60
	`
	rows, err := r.pool.Query(ctx, recentQuery, studentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetChildAttendanceSummary: recent: %w", err)
	}
	defer rows.Close()

	var recentPeriods []StudentAttendanceRecord
	for rows.Next() {
		var rec StudentAttendanceRecord
		if err := rows.Scan(&rec.Date, &rec.Subject, &rec.Status); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetChildAttendanceSummary: scan recent: %w", err)
		}
		recentPeriods = append(recentPeriods, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attendance.Repository.GetChildAttendanceSummary: rows: %w", err)
	}

	// Compute percentage on the fly if no summary exists yet
	percentage := summary.AttendancePercentage
	if err == pgx.ErrNoRows {
		// Compute from raw records
		countQuery := `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'PRESENT') AS present,
				COUNT(*) FILTER (WHERE status = 'ABSENT') AS absent,
				COUNT(*) FILTER (WHERE status = 'LATE') AS late,
				COUNT(*) FILTER (WHERE status = 'EXCUSED') AS excused
			FROM attendance_records
			WHERE student_id = $1 AND academic_term_id = $2 AND tenant_id = $3
		`
		var total, present, absent, late, excused int
		if err := r.pool.QueryRow(ctx, countQuery, studentID, termID, tenantID).Scan(&total, &present, &absent, &late, &excused); err != nil {
			return nil, fmt.Errorf("attendance.Repository.GetChildAttendanceSummary: compute: %w", err)
		}
		if total > 0 {
			percentage = float64(present) / float64(total) * 100
		}
	}

	return &ChildAttendanceSummary{
		StudentID:            studentID,
		TermID:               termID,
		AttendancePercentage: percentage,
		RecentPeriods:        recentPeriods,
	}, nil
}

func (r *pgRepository) ComputeTermSummaries(ctx context.Context, tenantID, schoolID, termID string) (int, error) {
	// Refresh materialised summaries for all students in this school/term
	query := `
		INSERT INTO attendance_term_summaries
			(tenant_id, school_id, student_id, academic_term_id, learning_area_id,
			 periods_total, periods_present, periods_absent, periods_late,
			 periods_excused, attendance_percentage, last_refreshed_at)
		SELECT
			$1 AS tenant_id,
			$2 AS school_id,
			ar.student_id,
			$3 AS academic_term_id,
			COALESCE(ts.learning_area_id, '00000000-0000-0000-0000-000000000000'::UUID) AS learning_area_id,
			COUNT(*) AS periods_total,
			COUNT(*) FILTER (WHERE ar.status = 'PRESENT') AS periods_present,
			COUNT(*) FILTER (WHERE ar.status = 'ABSENT') AS periods_absent,
			COUNT(*) FILTER (WHERE ar.status = 'LATE') AS periods_late,
			COUNT(*) FILTER (WHERE ar.status = 'EXCUSED') AS periods_excused,
			ROUND(
				(COUNT(*) FILTER (WHERE ar.status = 'PRESENT')::NUMERIC / NULLIF(COUNT(*), 0)) * 100,
				2
			) AS attendance_percentage,
			NOW() AS last_refreshed_at
		FROM attendance_records ar
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		WHERE ar.tenant_id = $1
		  AND ar.school_id = $2
		  AND ar.academic_term_id = $3
		GROUP BY ar.student_id, COALESCE(ts.learning_area_id, '00000000-0000-0000-0000-000000000000'::UUID)
		ON CONFLICT (student_id, academic_term_id, learning_area_id)
		DO UPDATE SET
			periods_total        = EXCLUDED.periods_total,
			periods_present      = EXCLUDED.periods_present,
			periods_absent       = EXCLUDED.periods_absent,
			periods_late         = EXCLUDED.periods_late,
			periods_excused      = EXCLUDED.periods_excused,
			attendance_percentage = EXCLUDED.attendance_percentage,
			last_refreshed_at    = NOW()
	`
	result, err := r.pool.Exec(ctx, query, tenantID, schoolID, termID)
	if err != nil {
		// Handle the potential UUID cast error for learning_area_id = NULL gracefully
		return 0, fmt.Errorf("attendance.Repository.ComputeTermSummaries: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// Ensure compile-time check that *pgRepository satisfies Repository.
var _ Repository = (*pgRepository)(nil)
