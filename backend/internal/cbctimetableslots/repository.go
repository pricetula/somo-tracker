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

	"somotracker/backend/internal/database"
)

// PgRepository handles timetable slot database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// List returns slots matching the given filter.
func (r *PgRepository) List(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error) {
	query := `
		SELECT id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at
		FROM cbc_timetable_slots
		WHERE academic_year_id = $1
	`
	args := []interface{}{filter.AcademicYearID}
	argIdx := 2

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
func (r *PgRepository) ListEnriched(ctx context.Context, filter SlotFilter) ([]SlotWithEnrichedData, error) {
	query := `
		SELECT
			sl.id,
			sl.academic_year_id,
			sl.structure_id,
			sl.class_id,
			sl.learning_area_id,
			sl.teacher_id,
			sl.room_identifier,
			sl.created_at,
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
		WHERE sl.academic_year_id = $1
	`
	args := []interface{}{filter.AcademicYearID}
	argIdx := 2

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
	}

	query += ` ORDER BY ts.day_of_week ASC, ts.start_time ASC, sl.class_id ASC`

	return r.queryEnrichedSlots(ctx, query, args...)
}

func (r *PgRepository) querySlots(ctx context.Context, query string, args ...interface{}) ([]TimetableSlot, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.querySlots: %w", err)
	}
	defer rows.Close()

	var slots []TimetableSlot
	for rows.Next() {
		var s TimetableSlot
		if err := rows.Scan(&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt); err != nil {
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

func (r *PgRepository) queryEnrichedSlots(ctx context.Context, query string, args ...interface{}) ([]SlotWithEnrichedData, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.queryEnrichedSlots: %w", err)
	}
	defer rows.Close()

	var slots []SlotWithEnrichedData
	for rows.Next() {
		var s SlotWithEnrichedData
		var startTime, endTime *time.Time
		if err := rows.Scan(
			&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID,
			&s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt,
			&s.ClassName, &s.PeriodName, &s.DayOfWeek,
			&startTime, &endTime, &s.IsBreak,
			&s.LearningAreaName, &s.TeacherName,
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
	const query = `
		SELECT id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at
		FROM cbc_timetable_slots
		WHERE id = $1
	`

	var s TimetableSlot
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt,
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
			sl.academic_year_id,
			sl.structure_id,
			sl.class_id,
			sl.learning_area_id,
			sl.teacher_id,
			sl.room_identifier,
			sl.created_at,
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
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID,
		&s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt,
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

// Create inserts a new slot. Returns the created slot.
func (r *PgRepository) Create(ctx context.Context, slot CreateSlotPayload) (*TimetableSlot, error) {
	const query = `
		INSERT INTO cbc_timetable_slots (academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at
	`

	var s TimetableSlot
	err := r.pool.QueryRow(ctx, query,
		slot.AcademicYearID, slot.StructureID, slot.ClassID,
		slot.LearningAreaID, slot.TeacherID, slot.RoomIdentifier,
	).Scan(&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("cbctimetableslots.Repository.Create: %w", mapUniqueViolation(err))
		}
		return nil, fmt.Errorf("cbctimetableslots.Repository.Create: %w", err)
	}
	return &s, nil
}

// BatchCreate inserts multiple slots atomically.
func (r *PgRepository) BatchCreate(ctx context.Context, slots []CreateSlotPayload) ([]TimetableSlot, error) {
	if len(slots) == 0 {
		return []TimetableSlot{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Repository.BatchCreate: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		INSERT INTO cbc_timetable_slots (academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at
	`

	var results []TimetableSlot
	for _, slot := range slots {
		var s TimetableSlot
		err := tx.QueryRow(ctx, query,
			slot.AcademicYearID, slot.StructureID, slot.ClassID,
			slot.LearningAreaID, slot.TeacherID, slot.RoomIdentifier,
		).Scan(&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt)
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
	// Build dynamic SET clause for optional fields
	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if slot.LearningAreaID != nil {
		sets = append(sets, fmt.Sprintf("learning_area_id = $%d", argIdx))
		args = append(args, *slot.LearningAreaID)
		argIdx++
	} else {
		sets = append(sets, fmt.Sprintf("learning_area_id = $%d", argIdx))
		args = append(args, nil)
		argIdx++
	}

	if slot.TeacherID != nil {
		sets = append(sets, fmt.Sprintf("teacher_id = $%d", argIdx))
		args = append(args, *slot.TeacherID)
		argIdx++
	} else {
		sets = append(sets, fmt.Sprintf("teacher_id = $%d", argIdx))
		args = append(args, nil)
		argIdx++
	}

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
		RETURNING id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier, created_at
	`, strings.Join(sets, ", "), argIdx)

	var s TimetableSlot
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.AcademicYearID, &s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID, &s.RoomIdentifier, &s.CreatedAt,
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
	tag, err := r.pool.Exec(ctx, query, id)
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

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("cbctimetableslots.Repository.ClearDay: %w", err)
	}
	return nil
}

// ClearClassDay removes all slots for a specific class on a given structure day.
func (r *PgRepository) ClearClassDay(ctx context.Context, structureID, classID string) error {
	const query = `DELETE FROM cbc_timetable_slots WHERE structure_id = $1 AND class_id = $2`
	_, err := r.pool.Exec(ctx, query, structureID, classID)
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
