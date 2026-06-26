package timetable

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles timetable database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// BulkUpsertSlots inserts or updates many timetable slots in a single statement.
// The DB trigger (trg_auto_register_subject_teacher) automatically registers
// subject teachers for slots that include a learning_area_id.
func (r *PgRepository) BulkUpsertSlots(ctx context.Context, tenantID, schoolID string, input BulkCreateTimetableSlotsInput) error {
	if len(input.Slots) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("timetable.Repository.BulkUpsertSlots: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const upsertSQL = `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, cbc_learning_area_id, room_identifier,
			 day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::time, $11::time)
		ON CONFLICT ON CONSTRAINT excl_cbc_timetable_teacher DO NOTHING
	`

	for _, slot := range input.Slots {
		_, err = tx.Exec(ctx, upsertSQL,
			tenantID, schoolID, input.AcademicYearID, input.AcademicTermID,
			slot.ClassID, slot.TeacherID, slot.LearningAreaID, slot.RoomIdentifier,
			slot.DayOfWeek, slot.StartTime, slot.EndTime,
		)
		if err != nil {
			return fmt.Errorf("timetable.Repository.BulkUpsertSlots: exec: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("timetable.Repository.BulkUpsertSlots: commit tx: %w", err)
	}

	return nil
}

// GetSlotsByClass returns all timetable slots for a given class and term.
func (r *PgRepository) GetSlotsByClass(ctx context.Context, tenantID, classID, termID string) ([]TimetableSlot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, academic_term_id,
		       class_id, teacher_id, cbc_learning_area_id, room_identifier,
		       day_of_week, start_time::text, end_time::text
		FROM cbc_timetable_slots
		WHERE tenant_id = $1
		  AND class_id = $2
		  AND academic_term_id = $3
		ORDER BY day_of_week, start_time
	`

	rows, err := r.pool.Query(ctx, query, tenantID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.GetSlotsByClass: %w", err)
	}
	defer rows.Close()

	return scanSlots(rows)
}

// GetSlotsByTeacher returns all timetable slots for a given teacher and term.
func (r *PgRepository) GetSlotsByTeacher(ctx context.Context, tenantID, teacherID, termID string) ([]TimetableSlot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, academic_term_id,
		       class_id, teacher_id, cbc_learning_area_id, room_identifier,
		       day_of_week, start_time::text, end_time::text
		FROM cbc_timetable_slots
		WHERE tenant_id = $1
		  AND teacher_id = $2
		  AND academic_term_id = $3
		ORDER BY day_of_week, start_time
	`

	rows, err := r.pool.Query(ctx, query, tenantID, teacherID, termID)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.GetSlotsByTeacher: %w", err)
	}
	defer rows.Close()

	return scanSlots(rows)
}

// AssignClassTeacher assigns a teacher to a class with the given role.
func (r *PgRepository) AssignClassTeacher(ctx context.Context, input ClassTeacherInput) error {
	const upsertSQL = `
		INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, teacher_role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (class_id, user_id, learning_area_id)
		DO UPDATE SET teacher_role = EXCLUDED.teacher_role
	`

	_, err := r.pool.Exec(ctx, upsertSQL,
		input.TenantID, input.ClassID, input.UserID,
		input.LearningAreaID, input.TeacherRole,
	)
	if err != nil {
		return fmt.Errorf("timetable.Repository.AssignClassTeacher: %w", err)
	}
	return nil
}

// RemoveClassTeacher removes a teacher assignment from a class.
func (r *PgRepository) RemoveClassTeacher(ctx context.Context, tenantID, classID, userID string) error {
	const deleteSQL = `
		DELETE FROM cbc_class_teachers
		WHERE tenant_id = $1 AND class_id = $2 AND user_id = $3
	`

	result, err := r.pool.Exec(ctx, deleteSQL, tenantID, classID, userID)
	if err != nil {
		return fmt.Errorf("timetable.Repository.RemoveClassTeacher: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("timetable.Repository.RemoveClassTeacher: %w", ErrNotFound)
	}
	return nil
}

// HasPrimaryRole checks if a teacher already holds PRIMARY_CLASS_TEACHER on any class.
func (r *PgRepository) HasPrimaryRole(ctx context.Context, tenantID, userID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_class_teachers
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND teacher_role = 'PRIMARY_CLASS_TEACHER'
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("timetable.Repository.HasPrimaryRole: %w", err)
	}
	return exists, nil
}

// ValidateTerm checks that the term exists and belongs to the tenant + school.
func (r *PgRepository) ValidateTerm(ctx context.Context, tenantID, schoolID, termID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM academic_terms
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, termID, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("timetable.Repository.ValidateTerm: %w", err)
	}
	return exists, nil
}

// scanSlots is a shared helper to scan timetable slot rows.
func scanSlots(rows pgx.Rows) ([]TimetableSlot, error) {
	var slots []TimetableSlot
	for rows.Next() {
		var s TimetableSlot
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID,
			&s.AcademicYearID, &s.AcademicTermID,
			&s.ClassID, &s.TeacherID,
			&s.LearningAreaID, &s.RoomIdentifier,
			&s.DayOfWeek, &s.StartTime, &s.EndTime,
		); err != nil {
			return nil, fmt.Errorf("timetable.scanSlots: %w", err)
		}
		slots = append(slots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetable.scanSlots: rows: %w", err)
	}
	if slots == nil {
		slots = []TimetableSlot{}
	}
	return slots, nil
}
