package cbctimetable

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository handles database operations for CBC timetable.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pools *database.Pools) *Repository {
	return &Repository{pool: pools.PG}
}

// ─── CRUD: Slots ──────────────────────────────────────────────────────────

// FetchSlotsByClass returns all timetable slots for a class in the current academic year.
func (r *Repository) FetchSlotsByClass(ctx context.Context, classID string) ([]TimetableSlot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, class_id, teacher_id,
		       cbc_learning_area_id, room_identifier, day_of_week, start_time, end_time
		FROM cbc_timetable_slots
		WHERE class_id = $1
		ORDER BY day_of_week, start_time ASC
	`
	rows, err := r.pool.Query(ctx, query, classID)
	if err != nil {
		return nil, fmt.Errorf("fetch slots by class: %w", err)
	}
	defer rows.Close()

	return scanSlots(rows)
}

// FetchSlotByID returns a single slot by ID.
func (r *Repository) FetchSlotByID(ctx context.Context, slotID string) (*TimetableSlot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, class_id, teacher_id,
		       cbc_learning_area_id, room_identifier, day_of_week, start_time, end_time
		FROM cbc_timetable_slots
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, slotID)
	slot, err := scanSlot(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch slot by id: %w", err)
	}
	return slot, nil
}

// CreateSlot inserts a new timetable slot.
func (r *Repository) CreateSlot(ctx context.Context, slot *TimetableSlot) error {
	const query = `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, class_id, teacher_id,
			 cbc_learning_area_id, room_identifier, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		slot.TenantID, slot.SchoolID, slot.AcademicYearID, slot.ClassID, slot.TeacherID,
		slot.LearningAreaID, slot.RoomIdentifier, slot.DayOfWeek, slot.StartTime, slot.EndTime,
	).Scan(&slot.ID)
	if err != nil {
		return fmt.Errorf("create slot: %w", err)
	}
	return nil
}

// UpdateSlot updates an existing timetable slot.
func (r *Repository) UpdateSlot(ctx context.Context, slot *TimetableSlot) error {
	const query = `
		UPDATE cbc_timetable_slots
		SET teacher_id = $1, cbc_learning_area_id = $2, room_identifier = $3,
		    day_of_week = $4, start_time = $5, end_time = $6
		WHERE id = $7
	`
	tag, err := r.pool.Exec(ctx, query,
		slot.TeacherID, slot.LearningAreaID, slot.RoomIdentifier,
		slot.DayOfWeek, slot.StartTime, slot.EndTime, slot.ID,
	)
	if err != nil {
		return fmt.Errorf("update slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("slot not found: %s", slot.ID)
	}
	return nil
}

// DeleteSlot removes a timetable slot by ID.
func (r *Repository) DeleteSlot(ctx context.Context, slotID string) error {
	const query = `DELETE FROM cbc_timetable_slots WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, slotID)
	if err != nil {
		return fmt.Errorf("delete slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("slot not found: %s", slotID)
	}
	return nil
}

// ─── Conflict detection ────────────────────────────────────────────────────

// FindTeacherOverlaps returns slots that overlap with the given teacher/time.
// Excludes a specific slot ID (for edit-in-place) and/or class ID if provided.
func (r *Repository) FindTeacherOverlaps(
	ctx context.Context,
	teacherID string,
	dayOfWeek int,
	startTime, endTime string,
	academicYearID string,
	schoolID string,
	excludeSlotID *string,
	excludeClassID *string,
) ([]conflictingSlot, error) {
	query := `
		SELECT ts.id, c.name AS class_name, ts.start_time, ts.end_time,
		       u.first_name || ' ' || u.last_name AS teacher_name
		FROM cbc_timetable_slots ts
		JOIN classes c ON c.id = ts.class_id AND c.tenant_id = ts.tenant_id
		JOIN users u ON u.id = ts.teacher_id
		WHERE ts.teacher_id = $1
		  AND ts.academic_year_id = $2
		  AND ts.school_id = $3
		  AND ts.day_of_week = $4
		  AND ts.start_time < $6
		  AND ts.end_time > $5
	`
	args := []any{teacherID, academicYearID, schoolID, dayOfWeek, startTime, endTime}
	argIdx := 7

	if excludeSlotID != nil && *excludeSlotID != "" {
		query += fmt.Sprintf(" AND ts.id != $%d", argIdx)
		args = append(args, *excludeSlotID)
		argIdx++
	}
	if excludeClassID != nil && *excludeClassID != "" {
		query += fmt.Sprintf(" AND ts.class_id != $%d", argIdx)
		args = append(args, *excludeClassID)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find teacher overlaps: %w", err)
	}
	defer rows.Close()

	return scanConflicts(rows)
}

// FindRoomOverlaps returns slots that overlap with the given room/time.
func (r *Repository) FindRoomOverlaps(
	ctx context.Context,
	roomIdentifier string,
	dayOfWeek int,
	startTime, endTime string,
	academicYearID string,
	schoolID string,
	excludeSlotID *string,
	excludeClassID *string,
) ([]conflictingSlot, error) {
	query := `
		SELECT ts.id, c.name AS class_name, ts.start_time, ts.end_time,
		       u.first_name || ' ' || u.last_name AS teacher_name
		FROM cbc_timetable_slots ts
		JOIN classes c ON c.id = ts.class_id AND c.tenant_id = ts.tenant_id
		JOIN users u ON u.id = ts.teacher_id
		WHERE ts.room_identifier = $1
		  AND ts.academic_year_id = $2
		  AND ts.school_id = $3
		  AND ts.day_of_week = $4
		  AND ts.start_time < $6
		  AND ts.end_time > $5
	`
	args := []any{roomIdentifier, academicYearID, schoolID, dayOfWeek, startTime, endTime}
	argIdx := 7

	if excludeSlotID != nil && *excludeSlotID != "" {
		query += fmt.Sprintf(" AND ts.id != $%d", argIdx)
		args = append(args, *excludeSlotID)
		argIdx++
	}
	if excludeClassID != nil && *excludeClassID != "" {
		query += fmt.Sprintf(" AND ts.class_id != $%d", argIdx)
		args = append(args, *excludeClassID)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find room overlaps: %w", err)
	}
	defer rows.Close()

	return scanConflicts(rows)
}

// ─── Bulk operations ───────────────────────────────────────────────────────

// FetchSlotsByClassAndDay returns all slots for a class on a specific day.
func (r *Repository) FetchSlotsByClassAndDay(
	ctx context.Context,
	classID string,
	dayOfWeek int,
	academicYearID string,
) ([]TimetableSlot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, class_id, teacher_id,
		       cbc_learning_area_id, room_identifier, day_of_week, start_time, end_time
		FROM cbc_timetable_slots
		WHERE class_id = $1 AND day_of_week = $2 AND academic_year_id = $3
		ORDER BY start_time ASC
	`
	rows, err := r.pool.Query(ctx, query, classID, dayOfWeek, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("fetch slots by class and day: %w", err)
	}
	defer rows.Close()

	return scanSlots(rows)
}

// BulkInsertSlots inserts multiple slots in a single batch (non-conflicting).
// Returns the count of successfully inserted slots.
func (r *Repository) BulkInsertSlots(
	ctx context.Context,
	slots []TimetableSlot,
) (int, error) {
	if len(slots) == 0 {
		return 0, nil
	}

	const query = `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, class_id, teacher_id,
			 cbc_learning_area_id, room_identifier, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	inserted := 0
	for _, slot := range slots {
		_, err := r.pool.Exec(ctx, query,
			slot.TenantID, slot.SchoolID, slot.AcademicYearID, slot.ClassID, slot.TeacherID,
			slot.LearningAreaID, slot.RoomIdentifier, slot.DayOfWeek, slot.StartTime, slot.EndTime,
		)
		if err != nil {
			// Skip slots that violate constraints; don't abort the batch
			continue
		}
		inserted++
	}
	return inserted, nil
}

// ─── Counts and metadata ───────────────────────────────────────────────────

// CountAttendancePeriodsForSlot returns how many attendance periods reference this slot's learning area for this class.
func (r *Repository) CountAttendancePeriodsForSlot(ctx context.Context, slotID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM cbc_attendance_periods ap
		JOIN cbc_timetable_slots ts ON ts.class_id = ap.class_id
			AND ts.cbc_learning_area_id = ap.cbc_learning_area_id
			AND ts.tenant_id = ap.tenant_id
		WHERE ts.id = $1
	`
	var count int
	err := r.pool.QueryRow(ctx, query, slotID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count attendance periods: %w", err)
	}
	return count, nil
}

// ─── Reference data ────────────────────────────────────────────────────────

// FetchLearningAreasByGrade returns learning areas for a given grade.
func (r *Repository) FetchLearningAreasByGrade(ctx context.Context, gradeID string) ([]LearningAreaBrief, error) {
	const query = `
		SELECT id, name, code
		FROM cbc_learning_areas
		WHERE grade_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, gradeID)
	if err != nil {
		return nil, fmt.Errorf("fetch learning areas: %w", err)
	}
	defer rows.Close()

	var areas []LearningAreaBrief
	for rows.Next() {
		var a LearningAreaBrief
		if err := rows.Scan(&a.ID, &a.Name, &a.Code); err != nil {
			return nil, fmt.Errorf("scan learning area: %w", err)
		}
		areas = append(areas, a)
	}
	return areas, nil
}

// FetchTeachersBySchool returns all teachers (users with TEACHER membership) for a school.
func (r *Repository) FetchTeachersBySchool(ctx context.Context, schoolID, tenantID string) ([]TeacherBrief, error) {
	const query = `
		SELECT u.id, u.first_name, u.last_name, u.email
		FROM users u
		JOIN memberships m ON m.user_id = u.id AND m.tenant_id = u.tenant_id
		WHERE m.school_id = $1 AND m.tenant_id = $2 AND m.role = 'TEACHER' AND m.is_active = true AND u.is_active = true
		ORDER BY u.first_name, u.last_name
	`
	rows, err := r.pool.Query(ctx, query, schoolID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch teachers: %w", err)
	}
	defer rows.Close()

	var teachers []TeacherBrief
	for rows.Next() {
		var t TeacherBrief
		if err := rows.Scan(&t.ID, &t.FirstName, &t.LastName, &t.Email); err != nil {
			return nil, fmt.Errorf("scan teacher: %w", err)
		}
		t.Name = t.FirstName + " " + t.LastName
		teachers = append(teachers, t)
	}
	return teachers, nil
}

// FetchClassTeachers returns teachers linked to a class (optionally filtered by learning area).
func (r *Repository) FetchClassTeachers(ctx context.Context, classID string, learningAreaID *string) ([]TeacherBrief, error) {
	query := `
		SELECT u.id, u.first_name, u.last_name, u.email
		FROM cbc_class_teachers ct
		JOIN users u ON u.id = ct.user_id
		WHERE ct.class_id = $1
	`
	args := []any{classID}

	if learningAreaID != nil && *learningAreaID != "" {
		query += ` AND ct.learning_area_id = $2`
		args = append(args, *learningAreaID)
	}

	query += ` ORDER BY u.first_name, u.last_name`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fetch class teachers: %w", err)
	}
	defer rows.Close()

	var teachers []TeacherBrief
	for rows.Next() {
		var t TeacherBrief
		if err := rows.Scan(&t.ID, &t.FirstName, &t.LastName, &t.Email); err != nil {
			return nil, fmt.Errorf("scan class teacher: %w", err)
		}
		t.Name = t.FirstName + " " + t.LastName
		teachers = append(teachers, t)
	}
	return teachers, nil
}

// FetchRoomAutocomplete returns previously used room identifiers matching a query.
func (r *Repository) FetchRoomAutocomplete(ctx context.Context, query string, schoolID, tenantID string) ([]string, error) {
	const sql = `
		SELECT DISTINCT room_identifier
		FROM cbc_timetable_slots
		WHERE school_id = $1 AND tenant_id = $2
		  AND room_identifier IS NOT NULL
		  AND room_identifier ILIKE $3
		ORDER BY room_identifier
		LIMIT 10
	`
	rows, err := r.pool.Query(ctx, sql, schoolID, tenantID, query+"%")
	if err != nil {
		return nil, fmt.Errorf("fetch room autocomplete: %w", err)
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}
		rooms = append(rooms, r)
	}
	return rooms, nil
}

// FetchClassBrief returns lightweight class info by ID.
func (r *Repository) FetchClassBrief(ctx context.Context, classID string) (*ClassBrief, error) {
	const query = `
		SELECT c.id, c.name, g.name AS grade_name, c.stream
		FROM classes c
		JOIN grades g ON g.id = c.grade_id
		WHERE c.id = $1
	`
	var b ClassBrief
	err := r.pool.QueryRow(ctx, query, classID).Scan(&b.ID, &b.Name, &b.GradeName, &b.Stream)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch class brief: %w", err)
	}
	return &b, nil
}

// FetchClassGradeID returns the grade_id for a class.
func (r *Repository) FetchClassGradeID(ctx context.Context, classID string) (string, error) {
	const query = `SELECT grade_id FROM classes WHERE id = $1`
	var gradeID string
	err := r.pool.QueryRow(ctx, query, classID).Scan(&gradeID)
	if err != nil {
		return "", fmt.Errorf("fetch class grade id: %w", err)
	}
	return gradeID, nil
}

// FetchCurrentAcademicTerm returns the current term for a school/tenant.
func (r *Repository) FetchCurrentAcademicTerm(ctx context.Context, schoolID, tenantID string) (string, error) {
	const query = `
		SELECT id FROM academic_terms
		WHERE school_id = $1 AND tenant_id = $2 AND is_current = true
		LIMIT 1
	`
	var termID string
	err := r.pool.QueryRow(ctx, query, schoolID, tenantID).Scan(&termID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("no current academic term found")
		}
		return "", fmt.Errorf("fetch current term: %w", err)
	}
	return termID, nil
}

// FetchClassStudents returns enrolled students for a class in a term.
func (r *Repository) FetchClassStudents(ctx context.Context, classID, termID string) ([]StudentAttendanceRow, error) {
	const query = `
		SELECT s.id, s.first_name, s.last_name, s.gender
		FROM student_enrollments e
		JOIN students s ON s.id = e.student_id AND s.tenant_id = e.tenant_id
		WHERE e.class_id = $1 AND e.academic_term_id = $2 AND e.status = 'ACTIVE'
		ORDER BY s.first_name, s.last_name
	`
	rows, err := r.pool.Query(ctx, query, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("fetch class students: %w", err)
	}
	defer rows.Close()

	var students []StudentAttendanceRow
	for rows.Next() {
		var s StudentAttendanceRow
		if err := rows.Scan(&s.StudentID, &s.FirstName, &s.LastName, &s.Gender); err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}
		s.StudentName = s.FirstName + " " + s.LastName
		students = append(students, s)
	}
	return students, nil
}

// ─── Attendance helpers ────────────────────────────────────────────────────

// StudentAttendanceRow is a student record for the attendance grid.
type StudentAttendanceRow struct {
	StudentID   string `json:"student_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	StudentName string `json:"student_name"`
	Gender      string `json:"gender"`
}

// FetchTodayTeacherSlots returns today's timetable slots for a teacher.
func (r *Repository) FetchTodayTeacherSlots(ctx context.Context, teacherID string) ([]SlotBrief, error) {
	// Day of week: PostgreSQL EXTRACT(DOW) returns 0=Sunday, 1=Monday, ..., 6=Saturday
	// We store 1=Monday through 7=Sunday. Convert: DOW=0→7, DOW=1→1, ..., DOW=6→6
	const query = `
		SELECT
			ts.id || '_' || ts.class_id AS period_id,
			COALESCE(la.name, 'Free period') AS learning_area_name,
			ts.start_time::text,
			ts.end_time::text
		FROM cbc_timetable_slots ts
		LEFT JOIN cbc_learning_areas la ON la.id = ts.cbc_learning_area_id
		WHERE ts.teacher_id = $1
		  AND ts.day_of_week = CASE WHEN EXTRACT(DOW FROM CURRENT_DATE) = 0 THEN 7
		                            ELSE EXTRACT(DOW FROM CURRENT_DATE)::int END
		ORDER BY ts.start_time ASC
	`
	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("fetch today teacher slots: %w", err)
	}
	defer rows.Close()

	var slots []SlotBrief
	for rows.Next() {
		var s SlotBrief
		if err := rows.Scan(&s.PeriodID, &s.LearningAreaName, &s.StartTime, &s.EndTime); err != nil {
			return nil, fmt.Errorf("scan slot brief: %w", err)
		}
		slots = append(slots, s)
	}
	return slots, nil
}

// ResolveSchoolID returns the school_id for a class.
func (r *Repository) ResolveSchoolID(ctx context.Context, classID string) (string, error) {
	query := `SELECT school_id FROM classes WHERE id = $1`
	var schoolID string
	err := r.pool.QueryRow(ctx, query, classID).Scan(&schoolID)
	if err != nil {
		return "", fmt.Errorf("resolve school for class %s: %w", classID, err)
	}
	return schoolID, nil
}

// ResolveAcademicYearID returns the academic_year_id for a class.
func (r *Repository) ResolveAcademicYearID(ctx context.Context, classID string) (string, error) {
	const query = `SELECT academic_year_id FROM classes WHERE id = $1`
	var yearID string
	err := r.pool.QueryRow(ctx, query, classID).Scan(&yearID)
	if err != nil {
		return "", fmt.Errorf("resolve academic year for class %s: %w", classID, err)
	}
	return yearID, nil
}

// FetchOperatingDays returns operating days for a school from settings.
// Defaults to Mon-Fri (days 1-5) if no setting is configured.
func (r *Repository) FetchOperatingDays(ctx context.Context, schoolID, tenantID string) ([]int, error) {
	// For now, return default 1-5 (Mon-Fri). In production, this would read from a
	// school_settings table. The frontend degrades gracefully.
	return []int{1, 2, 3, 4, 5}, nil
}

// ─── Scanner helpers ──────────────────────────────────────────────────────

func scanSlots(rows pgx.Rows) ([]TimetableSlot, error) {
	var slots []TimetableSlot
	for rows.Next() {
		var s TimetableSlot
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.ClassID, &s.TeacherID,
			&s.LearningAreaID, &s.RoomIdentifier, &s.DayOfWeek, &s.StartTime, &s.EndTime,
		); err != nil {
			return nil, fmt.Errorf("scan slot: %w", err)
		}
		slots = append(slots, s)
	}
	return slots, nil
}

func scanSlot(row pgx.Row) (*TimetableSlot, error) {
	var s TimetableSlot
	err := row.Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.ClassID, &s.TeacherID,
		&s.LearningAreaID, &s.RoomIdentifier, &s.DayOfWeek, &s.StartTime, &s.EndTime,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanConflicts(rows pgx.Rows) ([]conflictingSlot, error) {
	var conflicts []conflictingSlot
	for rows.Next() {
		var c conflictingSlot
		if err := rows.Scan(
			&c.ID, &c.ClassName, &c.StartTime, &c.EndTime, &c.TeacherName,
		); err != nil {
			return nil, fmt.Errorf("scan conflict: %w", err)
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — periods
// ═══════════════════════════════════════════════════════════════════════════

// CreateAttendancePeriod inserts a new attendance period.
func (r *Repository) CreateAttendancePeriod(ctx context.Context, p *CbcAttendancePeriod) error {
	const query = `
		INSERT INTO cbc_attendance_periods
			(tenant_id, school_id, academic_term_id, class_id, cbc_learning_area_id, date_recorded, recorded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		p.TenantID, p.SchoolID, p.AcademicTermID, p.ClassID, p.LearningAreaID, p.DateRecorded, p.RecordedBy,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return fmt.Errorf("create attendance period: %w", err)
	}
	return nil
}

// FetchAttendancePeriodsByDate returns periods for a class on a given date.
func (r *Repository) FetchAttendancePeriodsByDate(ctx context.Context, classID, date string) ([]CbcAttendancePeriod, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_term_id, class_id,
		       cbc_learning_area_id, date_recorded::text, recorded_by, created_at::text
		FROM cbc_attendance_periods
		WHERE class_id = $1 AND date_recorded = $2
		ORDER BY cbc_learning_area_id
	`
	rows, err := r.pool.Query(ctx, query, classID, date)
	if err != nil {
		return nil, fmt.Errorf("fetch periods by date: %w", err)
	}
	defer rows.Close()
	return scanPeriods(rows)
}

// FetchAttendancePeriodSummaries returns period summaries for a class in a date range.
func (r *Repository) FetchAttendancePeriodSummaries(ctx context.Context, classID, from, to string) ([]AttendancePeriodSummary, error) {
	const query = `
		SELECT
			ap.id,
			ap.date_recorded::text,
			ap.cbc_learning_area_id,
			COALESCE(la.name, '') AS learning_area_name,
			u.first_name || ' ' || u.last_name AS recorded_by_name,
			ap.recorded_by,
			ap.created_at::text,
			COALESCE(stats.total_students, 0)::int AS total_students,
			COALESCE(stats.present_count, 0)::int AS present_count,
			COALESCE(stats.absent_count, 0)::int AS absent_count,
			COALESCE(stats.late_count, 0)::int AS late_count,
			COALESCE(stats.excused_count, 0)::int AS excused_count,
			COALESCE(stats.unmarked_count, 0)::int AS unmarked_count
		FROM cbc_attendance_periods ap
		JOIN cbc_learning_areas la ON la.id = ap.cbc_learning_area_id
		JOIN users u ON u.id = ap.recorded_by
		LEFT JOIN LATERAL (
			SELECT
				COUNT(DISTINCT e.student_id) AS total_students,
				COUNT(l.id) FILTER (WHERE l.status = 'PRESENT') AS present_count,
				COUNT(l.id) FILTER (WHERE l.status = 'ABSENT') AS absent_count,
				COUNT(l.id) FILTER (WHERE l.status = 'LATE') AS late_count,
				COUNT(l.id) FILTER (WHERE l.status = 'EXCUSED') AS excused_count,
				(COUNT(DISTINCT e.student_id) - COUNT(l.id)) AS unmarked_count
			FROM student_enrollments e
			LEFT JOIN cbc_attendance_logs l
				ON l.student_id = e.student_id
				AND l.cbc_attendance_period_id = ap.id
				AND l.tenant_id = e.tenant_id
			WHERE e.class_id = ap.class_id
			  AND e.academic_term_id = ap.academic_term_id
			  AND e.status = 'ACTIVE'
		) stats ON true
		WHERE ap.class_id = $1
		  AND ap.date_recorded >= $2
		  AND ap.date_recorded <= $3
		ORDER BY ap.date_recorded DESC, la.name ASC
	`
	rows, err := r.pool.Query(ctx, query, classID, from, to)
	if err != nil {
		return nil, fmt.Errorf("fetch period summaries: %w", err)
	}
	defer rows.Close()
	return scanPeriodSummaries(rows)
}

// FetchAttendancePeriodByID returns a single period by ID.
func (r *Repository) FetchAttendancePeriodByID(ctx context.Context, periodID string) (*CbcAttendancePeriod, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_term_id, class_id,
		       cbc_learning_area_id, date_recorded::text, recorded_by, created_at::text
		FROM cbc_attendance_periods
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, periodID)
	p, err := scanPeriod(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch period by id: %w", err)
	}
	return p, nil
}

// FetchAttendancePeriodSummary returns a single period summary.
func (r *Repository) FetchAttendancePeriodSummary(ctx context.Context, periodID string) (*AttendancePeriodSummary, error) {
	const query = `
		SELECT
			ap.id,
			ap.date_recorded::text,
			ap.cbc_learning_area_id,
			COALESCE(la.name, '') AS learning_area_name,
			u.first_name || ' ' || u.last_name AS recorded_by_name,
			ap.recorded_by,
			ap.created_at::text,
			COALESCE(stats.total_students, 0)::int AS total_students,
			COALESCE(stats.present_count, 0)::int AS present_count,
			COALESCE(stats.absent_count, 0)::int AS absent_count,
			COALESCE(stats.late_count, 0)::int AS late_count,
			COALESCE(stats.excused_count, 0)::int AS excused_count,
			COALESCE(stats.unmarked_count, 0)::int AS unmarked_count
		FROM cbc_attendance_periods ap
		JOIN cbc_learning_areas la ON la.id = ap.cbc_learning_area_id
		JOIN users u ON u.id = ap.recorded_by
		LEFT JOIN LATERAL (
			SELECT
				COUNT(DISTINCT e.student_id) AS total_students,
				COUNT(l.id) FILTER (WHERE l.status = 'PRESENT') AS present_count,
				COUNT(l.id) FILTER (WHERE l.status = 'ABSENT') AS absent_count,
				COUNT(l.id) FILTER (WHERE l.status = 'LATE') AS late_count,
				COUNT(l.id) FILTER (WHERE l.status = 'EXCUSED') AS excused_count,
				(COUNT(DISTINCT e.student_id) - COUNT(l.id)) AS unmarked_count
			FROM student_enrollments e
			LEFT JOIN cbc_attendance_logs l
				ON l.student_id = e.student_id
				AND l.cbc_attendance_period_id = ap.id
				AND l.tenant_id = e.tenant_id
			WHERE e.class_id = ap.class_id
			  AND e.academic_term_id = ap.academic_term_id
			  AND e.status = 'ACTIVE'
		) stats ON true
		WHERE ap.id = $1
	`
	row := r.pool.QueryRow(ctx, query, periodID)
	s, err := scanPeriodSummary(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch period summary: %w", err)
	}
	return s, nil
}

// FetchClassEnrolledCount returns the number of actively enrolled students.
func (r *Repository) FetchClassEnrolledCount(ctx context.Context, classID, termID string) (int, error) {
	const query = `
		SELECT COUNT(*) FROM student_enrollments
		WHERE class_id = $1 AND academic_term_id = $2 AND status = 'ACTIVE'
	`
	var count int
	err := r.pool.QueryRow(ctx, query, classID, termID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count enrolled students: %w", err)
	}
	return count, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — logs
// ═══════════════════════════════════════════════════════════════════════════

// FetchAttendanceLogsByPeriod returns all logs for a period with recorder details.
func (r *Repository) FetchAttendanceLogsByPeriod(ctx context.Context, periodID string) ([]AttendanceLogDetail, error) {
	const query = `
		SELECT
			l.id, l.tenant_id, l.cbc_attendance_period_id, l.student_id,
			l.status, l.remarks, l.recorded_by,
			u.first_name, u.last_name
		FROM cbc_attendance_logs l
		JOIN users u ON u.id = l.recorded_by
		WHERE l.cbc_attendance_period_id = $1
		ORDER BY l.student_id
	`
	rows, err := r.pool.Query(ctx, query, periodID)
	if err != nil {
		return nil, fmt.Errorf("fetch logs by period: %w", err)
	}
	defer rows.Close()

	var logs []AttendanceLogDetail
	for rows.Next() {
		var d AttendanceLogDetail
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.PeriodID, &d.StudentID,
			&d.Status, &d.Remarks, &d.RecordedBy,
			&d.RecorderFirstName, &d.RecorderLastName,
		); err != nil {
			return nil, fmt.Errorf("scan log detail: %w", err)
		}
		d.RecordedByLabel = d.RecorderFirstName + " " + d.RecorderLastName
		logs = append(logs, d)
	}
	return logs, nil
}

// UpsertAttendanceLog inserts or updates a single attendance log.
// Returns the log with its ID populated (new or existing).
func (r *Repository) UpsertAttendanceLog(ctx context.Context, log *CbcAttendanceLog) error {
	const query = `
		INSERT INTO cbc_attendance_logs
			(tenant_id, cbc_attendance_period_id, student_id, status, remarks, recorded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cbc_attendance_period_id, student_id)
		DO UPDATE SET status = EXCLUDED.status, remarks = EXCLUDED.remarks, recorded_by = EXCLUDED.recorded_by
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		log.TenantID, log.PeriodID, log.StudentID, log.Status, log.Remarks, log.RecordedBy,
	).Scan(&log.ID)
	if err != nil {
		return fmt.Errorf("upsert attendance log: %w", err)
	}
	return nil
}

// BatchUpsertAttendanceLogs inserts or updates multiple logs in a single batch.
func (r *Repository) BatchUpsertAttendanceLogs(ctx context.Context, tenantID, periodID, recordedBy string, marks []BatchLogMark) ([]CbcAttendanceLog, error) {
	if len(marks) == 0 {
		return nil, nil
	}

	const query = `
		INSERT INTO cbc_attendance_logs
			(tenant_id, cbc_attendance_period_id, student_id, status, remarks, recorded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cbc_attendance_period_id, student_id)
		DO UPDATE SET status = EXCLUDED.status, remarks = EXCLUDED.remarks, recorded_by = EXCLUDED.recorded_by
		RETURNING id, tenant_id, cbc_attendance_period_id, student_id, status, remarks, recorded_by
	`

	logs := make([]CbcAttendanceLog, 0, len(marks))
	for _, m := range marks {
		var log CbcAttendanceLog
		err := r.pool.QueryRow(ctx, query,
			tenantID, periodID, m.StudentID, m.Status, m.Remarks, recordedBy,
		).Scan(&log.ID, &log.TenantID, &log.PeriodID, &log.StudentID, &log.Status, &log.Remarks, &log.RecordedBy)
		if err != nil {
			return nil, fmt.Errorf("batch upsert log for student %s: %w", m.StudentID, err)
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — analytics
// ═══════════════════════════════════════════════════════════════════════════

// FetchAttendanceHeatmap returns per-day attendance stats for a class/term.
func (r *Repository) FetchAttendanceHeatmap(ctx context.Context, classID, termID string) ([]AttendanceHeatmapDay, error) {
	const query = `
		SELECT
			ap.date_recorded::text AS date,
			COUNT(DISTINCT ap.id)::int AS period_count,
			COUNT(l.id)::int AS total_marks,
			COUNT(l.id) FILTER (WHERE l.status = 'PRESENT')::int AS present_count
		FROM cbc_attendance_periods ap
		JOIN cbc_attendance_logs l ON l.cbc_attendance_period_id = ap.id AND l.tenant_id = ap.tenant_id
		WHERE ap.class_id = $1 AND ap.academic_term_id = $2
		GROUP BY ap.date_recorded
		ORDER BY ap.date_recorded ASC
	`
	rows, err := r.pool.Query(ctx, query, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("fetch attendance heatmap: %w", err)
	}
	defer rows.Close()

	var days []AttendanceHeatmapDay
	for rows.Next() {
		var d AttendanceHeatmapDay
		var presentCount int
		if err := rows.Scan(&d.Date, &d.PeriodCount, &d.TotalMarks, &presentCount); err != nil {
			return nil, fmt.Errorf("scan heatmap day: %w", err)
		}
		if d.TotalMarks > 0 {
			rate := float64(presentCount) / float64(d.TotalMarks) * 100
			d.PresentRate = &rate
		}
		days = append(days, d)
	}
	return days, nil
}

// FetchAttendanceGaps returns timetable slots with no attendance period for the given dates.
func (r *Repository) FetchAttendanceGaps(ctx context.Context, classID, from, to string) ([]AttendanceGap, error) {
	// Generate all dates in range that match each slot's day_of_week,
	// then exclude those with an existing attendance period.
	// PostgreSQL EXTRACT(DOW): 0=Sun...6=Sat → we store 1=Mon...7=Sun
	const query = `
		WITH date_range AS (
			SELECT d::date AS date,
			       CASE WHEN EXTRACT(DOW FROM d) = 0 THEN 7
			            ELSE EXTRACT(DOW FROM d)::int END AS dow
			FROM generate_series($2::date, $3::date, '1 day'::interval) d
		),
		slot_dates AS (
			SELECT ts.id AS slot_id, ts.class_id, ts.cbc_learning_area_id,
			       COALESCE(la.name, 'Free Period') AS learning_area_name,
			       ts.day_of_week, ts.start_time::text, ts.end_time::text,
			       dr.date::text
			FROM cbc_timetable_slots ts
			JOIN date_range dr ON dr.dow = ts.day_of_week
			LEFT JOIN cbc_learning_areas la ON la.id = ts.cbc_learning_area_id
			WHERE ts.class_id = $1
		)
		SELECT sd.slot_id, sd.class_id, sd.cbc_learning_area_id,
		       sd.learning_area_name, sd.day_of_week, sd.start_time, sd.end_time, sd.date
		FROM slot_dates sd
		LEFT JOIN cbc_attendance_periods ap
			ON ap.class_id = sd.class_id
			AND ap.cbc_learning_area_id = sd.cbc_learning_area_id
			AND ap.date_recorded = sd.date::date
		WHERE ap.id IS NULL
		ORDER BY sd.date, sd.start_time
	`
	rows, err := r.pool.Query(ctx, query, classID, from, to)
	if err != nil {
		return nil, fmt.Errorf("fetch attendance gaps: %w", err)
	}
	defer rows.Close()

	var gaps []AttendanceGap
	for rows.Next() {
		var g AttendanceGap
		if err := rows.Scan(
			&g.SlotID, &g.ClassID, &g.LearningAreaID,
			&g.LearningAreaName, &g.DayOfWeek, &g.StartTime, &g.EndTime, &g.Date,
		); err != nil {
			return nil, fmt.Errorf("scan attendance gap: %w", err)
		}
		gaps = append(gaps, g)
	}
	return gaps, nil
}

// ─── Scanner helpers for attendance ──────────────────────────────────────

func scanPeriods(rows pgx.Rows) ([]CbcAttendancePeriod, error) {
	var periods []CbcAttendancePeriod
	for rows.Next() {
		p, err := scanPeriod(rows)
		if err != nil {
			return nil, err
		}
		periods = append(periods, *p)
	}
	return periods, nil
}

func scanPeriod(row pgx.Row) (*CbcAttendancePeriod, error) {
	var p CbcAttendancePeriod
	err := row.Scan(
		&p.ID, &p.TenantID, &p.SchoolID, &p.AcademicTermID, &p.ClassID,
		&p.LearningAreaID, &p.DateRecorded, &p.RecordedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPeriodSummaries(rows pgx.Rows) ([]AttendancePeriodSummary, error) {
	var summaries []AttendancePeriodSummary
	for rows.Next() {
		s, err := scanPeriodSummary(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, *s)
	}
	return summaries, nil
}

func scanPeriodSummary(row pgx.Row) (*AttendancePeriodSummary, error) {
	var s AttendancePeriodSummary
	err := row.Scan(
		&s.ID, &s.DateRecorded, &s.LearningAreaID, &s.LearningAreaName,
		&s.RecordedByName, &s.RecordedByID, &s.RecordedAt,
		&s.TotalStudents, &s.PresentCount, &s.AbsentCount,
		&s.LateCount, &s.ExcusedCount, &s.UnmarkedCount,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

var _ = strings.Join
