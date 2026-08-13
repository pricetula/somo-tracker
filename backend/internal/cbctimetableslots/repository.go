package cbctimetableslots

import (
	"context"
	"errors"
	"fmt"

	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// PgRepository handles timetable slot database operations.
type PgRepository struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools, logger *zap.SugaredLogger) *PgRepository {
	return &PgRepository{pool: pools.PG, logger: logger}
}

// slotColumns is the shared column list for SELECT queries on cbc_timetable_slots.
const slotColumns = `id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at, updated_at`

// List returns slots matching the given filter.
func (r *PgRepository) List(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM cbc_timetable_slots
		WHERE academic_year_id = $1
	`, slotColumns)
	args := []interface{}{filter.AcademicYearID}
	argIdx := 2

	if filter.TenantID != "" {
		query += fmt.Sprintf(` AND tenant_id = $%d`, argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.SchoolID != "" {
		query += fmt.Sprintf(` AND school_id = $%d`, argIdx)
		args = append(args, filter.SchoolID)
		argIdx++
	}
	if filter.StructureID != "" {
		query += fmt.Sprintf(` AND structure_id = $%d`, argIdx)
		args = append(args, filter.StructureID)
		argIdx++
	}
	if filter.ClassID != "" {
		query += fmt.Sprintf(` AND class_id = $%d`, argIdx)
		args = append(args, filter.ClassID)
		argIdx++
	}
	if filter.TeacherID != "" {
		query += fmt.Sprintf(` AND teacher_id = $%d`, argIdx)
		args = append(args, filter.TeacherID)
		argIdx++
	}
	if filter.RoomIdentifier != "" {
		query += fmt.Sprintf(` AND room_identifier = $%d`, argIdx)
		args = append(args, filter.RoomIdentifier)
	}

	query += ` ORDER BY created_at ASC`

	return r.querySlots(ctx, query, args...)
}

// ListEnriched returns slots with joined data from timetable_structures, classes, teachers.
// When filter.Date is set, the query filters by day-of-week matching that date and LEFT JOINs
// cbc_attendance_sessions to include session_status and skip_reason.
// The session_status and skip_reason columns are always selected (NULL when no date filter).
func (r *PgRepository) ListEnriched(ctx context.Context, filter SlotFilter) ([]SlotWithEnrichedData, error) {
	var hasDate bool
	selectCols := `
		sl.id,
		sl.tenant_id,
		sl.school_id,
		sl.academic_year_id,
		sl.structure_id,
		sl.class_id,
		sl.learning_area_id,
		sl.teacher_id,
		sl.room_identifier,
		sl.created_at,
		sl.updated_at,
		cls.grade_level || ' ' || COALESCE(str.name, '') AS class_name,
		ts.period_name,
		ts.day_of_week,
		ts.start_time,
		ts.end_time,
		ts.is_break,
		la.name AS learning_area_name,
		u.full_name AS teacher_name,
		sess.status AS session_status,
		sess.skip_reason
	`
	if filter.Date != "" {
		hasDate = true
	}

	// ── Step 1: Build FROM and JOINs (no WHERE yet) ──
	query := `SELECT` + selectCols + `FROM cbc_timetable_slots sl
		JOIN timetable_structures ts ON ts.id = sl.structure_id
		LEFT JOIN cbc_classes cls ON cls.id = sl.class_id
		LEFT JOIN cbc_streams str ON str.id = cls.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = sl.learning_area_id
		LEFT JOIN users u ON u.id = sl.teacher_id
	`
	args := []interface{}{}
	argIdx := 1

	// ── Step 2: LEFT JOIN cbc_attendance_sessions (always present, with date filter or without) ──
	// Without a date, we still LEFT JOIN but the WHERE clause won't match any session rows
	// (or the ON condition acts as a cross-check). We use a lateral approach:
	// when no date is given, the JOIN matches nothing → sess.* are NULL.
	// When date is given, the JOIN filters by date and matches the row.
	if hasDate {
		query += fmt.Sprintf(`
			LEFT JOIN cbc_attendance_sessions sess
				ON sess.timetable_slot_id = sl.id
				AND sess.date = $%d::DATE
				AND sess.tenant_id = sl.tenant_id
		`, argIdx)
		args = append(args, filter.Date)
		argIdx++
	} else {
		query += `
			LEFT JOIN cbc_attendance_sessions sess
				ON FALSE
		`
	}

	// ── Step 3: WHERE clause ──
	query += fmt.Sprintf(` WHERE sl.academic_year_id = $%d`, argIdx)
	args = append(args, filter.AcademicYearID)
	argIdx++

	// ── Step 4: Optional filters ──
	if filter.TenantID != "" {
		query += fmt.Sprintf(` AND sl.tenant_id = $%d`, argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.SchoolID != "" {
		query += fmt.Sprintf(` AND sl.school_id = $%d`, argIdx)
		args = append(args, filter.SchoolID)
		argIdx++
	}
	if filter.StructureID != "" {
		query += fmt.Sprintf(` AND sl.structure_id = $%d`, argIdx)
		args = append(args, filter.StructureID)
		argIdx++
	}
	if filter.ClassID != "" {
		query += fmt.Sprintf(` AND sl.class_id = $%d`, argIdx)
		args = append(args, filter.ClassID)
		argIdx++
	}
	if filter.TeacherID != "" {
		query += fmt.Sprintf(` AND sl.teacher_id = $%d`, argIdx)
		args = append(args, filter.TeacherID)
		argIdx++
	}

	// ── Step 5: Day-of-week filter (also uses the date) ──
	if hasDate {
		// EXTRACT(DOW) returns 0=Sunday … 6=Saturday. Our day_of_week is 1=Monday … 7=Sunday.
		query += fmt.Sprintf(`
			AND ts.day_of_week = (
				SELECT CASE EXTRACT(DOW FROM $%d::DATE)::INT
					WHEN 0 THEN 7  -- Postgres Sunday → our Sunday (7)
					ELSE EXTRACT(DOW FROM $%d::DATE)::INT
				END
			)
		`, argIdx, argIdx)
		args = append(args, filter.Date)
	}

	query += ` ORDER BY ts.day_of_week ASC, ts.start_time ASC, sl.class_id ASC`

	return r.queryEnrichedSlots(ctx, query, hasDate, args...)
}

func (r *PgRepository) querySlots(ctx context.Context, query string, args ...interface{}) ([]TimetableSlot, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.querySlots: %w", err)
	}
	defer rows.Close()

	var slots []TimetableSlot
	for rows.Next() {
		var s TimetableSlot
		if err := rows.Scan(&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("cbctimetableslots.Repository.querySlots: scan: %w", err)
		}
		slots = append(slots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.querySlots: rows: %w", err)
	}

	if slots == nil {
		slots = []TimetableSlot{}
	}

	return slots, nil
}

func (r *PgRepository) queryEnrichedSlots(ctx context.Context, query string, hasDate bool, args ...interface{}) ([]SlotWithEnrichedData, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.queryEnrichedSlots: %w", err)
	}
	defer rows.Close()

	var slots []SlotWithEnrichedData
	for rows.Next() {
		var s SlotWithEnrichedData
		var startTime, endTime *time.Time

		// Always scan session_status and skip_reason — they are always in the SELECT now.
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID,
			&s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
			&s.ClassName, &s.PeriodName, &s.DayOfWeek,
			&startTime, &endTime, &s.IsBreak,
			&s.LearningAreaName, &s.TeacherName,
			&s.SessionStatus, &s.SkipReason,
		); err != nil {
			return nil, fmt.Errorf("cbctimetableslots.Repository.queryEnrichedSlots: scan: %w", err)
		}

		if startTime != nil {
			s.StartTime = startTime.Format("15:04")
		}
		if endTime != nil {
			s.EndTime = endTime.Format("15:04")
		}
		slots = append(slots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.queryEnrichedSlots: rows: %w", err)
	}

	if slots == nil {
		slots = []SlotWithEnrichedData{}
	}

	return slots, nil
}

// GetByID retrieves a single slot by ID.
func (r *PgRepository) GetByID(ctx context.Context, id string) (*TimetableSlot, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM cbc_timetable_slots
		WHERE id = $1
	`, slotColumns)

	var s TimetableSlot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbctimetableslots.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbctimetableslots.Repository.GetByID: %w", err)
	}
	return &s, nil
}

// GetEnrichedByID retrieves a single slot with joined data.
func (r *PgRepository) GetEnrichedByID(ctx context.Context, id string) (*SlotWithEnrichedData, error) {
	const query = `
		SELECT
			sl.id,
			sl.tenant_id,
			sl.school_id,
			sl.academic_year_id,
			sl.structure_id,
			sl.class_id,
			sl.learning_area_id,
			sl.teacher_id,
			sl.room_identifier,
			sl.created_at,
			sl.updated_at,
			cls.grade_level || ' ' || COALESCE(str.name, '') AS class_name,
			ts.period_name,
			ts.day_of_week,
			ts.start_time,
			ts.end_time,
			ts.is_break,
			la.name AS learning_area_name,
			u.full_name AS teacher_name
		FROM cbc_timetable_slots sl
		JOIN timetable_structures ts ON ts.id = sl.structure_id
		LEFT JOIN cbc_classes cls ON cls.id = sl.class_id
		LEFT JOIN cbc_streams str ON str.id = cls.stream_id
		LEFT JOIN cbc_learning_areas la ON la.id = sl.learning_area_id
		LEFT JOIN users u ON u.id = sl.teacher_id
		WHERE sl.id = $1
	`

	var s SlotWithEnrichedData
	var startTime, endTime *time.Time
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID,
		&s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
		&s.ClassName, &s.PeriodName, &s.DayOfWeek,
		&startTime, &endTime, &s.IsBreak,
		&s.LearningAreaName, &s.TeacherName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbctimetableslots.Repository.GetEnrichedByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbctimetableslots.Repository.GetEnrichedByID: %w", err)
	}
	if startTime != nil {
		s.StartTime = startTime.Format("15:04")
	}
	if endTime != nil {
		s.EndTime = endTime.Format("15:04")
	}
	return &s, nil
}

// Create inserts a new slot with tenant/school scoping. Returns the created slot.
func (r *PgRepository) Create(ctx context.Context, tenantID, schoolID string, slot CreateSlotPayload) (*TimetableSlot, error) {
	const query = `
		INSERT INTO cbc_timetable_slots (tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at, updated_at
	`

	var s TimetableSlot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		tenantID, schoolID, slot.AcademicYearID, slot.StructureID, slot.ClassID,
		slot.LearningAreaID, slot.TeacherID, slot.RoomIdentifier,
	).Scan(&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("cbctimetableslots.Repository.Create: %w", mapUniqueViolation(err))
		}
		return nil, fmt.Errorf("cbctimetableslots.Repository.Create: %w", err)
	}
	return &s, nil
}

// BatchCreate inserts multiple slots atomically within a transaction.
func (r *PgRepository) BatchCreate(ctx context.Context, tenantID, schoolID string, slots []CreateSlotPayload) ([]TimetableSlot, error) {
	if len(slots) == 0 {
		return []TimetableSlot{}, nil
	}

	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.BatchCreate: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			r.logger.Warnw("cbctimetableslots.Repository.BatchCreate: rollback error",
				"error", rbErr.Error(),
			)
		}
	}()

	const query = `
		INSERT INTO cbc_timetable_slots (tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at, updated_at
	`

	var results []TimetableSlot
	for _, slot := range slots {
		var s TimetableSlot
		err := tx.QueryRow(ctx, query,
			tenantID, schoolID, slot.AcademicYearID, slot.StructureID, slot.ClassID,
			slot.LearningAreaID, slot.TeacherID, slot.RoomIdentifier,
		).Scan(&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("cbctimetableslots.Repository.BatchCreate: %w", mapUniqueViolation(err))
			}
			return nil, fmt.Errorf("cbctimetableslots.Repository.BatchCreate: %w", err)
		}
		results = append(results, s)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.BatchCreate: commit: %w", err)
	}

	return results, nil
}

// Update modifies a slot's assignments.
func (r *PgRepository) Update(ctx context.Context, id string, slot UpdateSlotPayload) (*TimetableSlot, error) {
	// Build dynamic SET clause — learning_area_id and teacher_id are required,
	// room_identifier is optional (nil = keep existing, non-nil = set value).
	sets := []string{
		"learning_area_id = $1",
		"teacher_id = $2",
	}
	args := []interface{}{slot.LearningAreaID, slot.TeacherID}
	argIdx := 3

	if slot.RoomIdentifier != nil {
		sets = append(sets, fmt.Sprintf("room_identifier = $%d", argIdx))
		args = append(args, *slot.RoomIdentifier)
		argIdx++
	} else {
		sets = append(sets, fmt.Sprintf("room_identifier = $%d", argIdx))
		args = append(args, nil)
		argIdx++
	}

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE cbc_timetable_slots
		SET %s
		WHERE id = $%d
		RETURNING id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at, updated_at
	`, strings.Join(sets, ", "), argIdx)

	var s TimetableSlot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbctimetableslots.Repository.Update: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("cbctimetableslots.Repository.Update: %w", mapUniqueViolation(err))
		}
		return nil, fmt.Errorf("cbctimetableslots.Repository.Update: %w", err)
	}
	return &s, nil
}

// Delete removes a slot by ID.
func (r *PgRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM cbc_timetable_slots WHERE id = $1`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("cbctimetableslots.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cbctimetableslots.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// ClearDay removes all slots for the given structure IDs.
func (r *PgRepository) ClearDay(ctx context.Context, structureIDs []string) error {
	if len(structureIDs) == 0 {
		return nil
	}

	args := make([]interface{}, len(structureIDs))
	for i, id := range structureIDs {
		args[i] = id
	}

	placeholders := make([]string, len(structureIDs))
	for i := range structureIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		`DELETE FROM cbc_timetable_slots WHERE structure_id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	_, err := database.FromContext(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("cbctimetableslots.Repository.ClearDay: %w", err)
	}
	return nil
}

// ClearClassDay removes all slots for a specific class on a given structure day.
func (r *PgRepository) ClearClassDay(ctx context.Context, structureID, classID string) error {
	const query = `DELETE FROM cbc_timetable_slots WHERE structure_id = $1 AND class_id = $2`
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, query, structureID, classID)
	if err != nil {
		return fmt.Errorf("cbctimetableslots.Repository.ClearClassDay: %w", err)
	}
	return nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// mapUniqueViolation maps the constraint name to the appropriate sentinel error.
func mapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "unique_class_slot":
			return ErrClassSlotOccupied
		case "unique_teacher_slot":
			return ErrTeacherDoubleBooked
		case "unique_room_slot":
			return ErrRoomDoubleBooked
		}
	}
	return ErrConflict
}

// compile-time interface check
var _ Repository = (*PgRepository)(nil)
