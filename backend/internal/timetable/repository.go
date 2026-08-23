package timetable

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

type PgRepository struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

func NewRepository(pools *database.Pools, logger *zap.SugaredLogger) *PgRepository {
	return &PgRepository{pool: pools.PG, logger: logger}
}

func (r *PgRepository) ListBlocks(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error) {
	const query = `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break,
		       academic_year_id, COALESCE(order_index, 0) as order_index,
		       created_at, updated_at
		FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3
		ORDER BY day_of_week ASC, order_index ASC, start_time ASC
	`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListBlocks: %w", err)
	}
	defer rows.Close()

	var blocks []TimeBlock
	for rows.Next() {
		var b TimeBlock
		if err := rows.Scan(
			&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
			&b.IsBreak, &b.AcademicYearID, &b.Order,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("timetable.Repository.ListBlocks: scan: %w", err)
		}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListBlocks: rows: %w", err)
	}

	if blocks == nil {
		blocks = []TimeBlock{}
	}
	return blocks, nil
}

func (r *PgRepository) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	const query = `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break,
		       academic_year_id, COALESCE(order_index, 0) as order_index,
		       created_at, updated_at
		FROM timetable_structures
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.AcademicYearID, &b.Order,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetable.Repository.GetBlock: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("timetable.Repository.GetBlock: %w", err)
	}
	return &b, nil
}

func (r *PgRepository) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	const query = `
		INSERT INTO timetable_structures (tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break,
		          academic_year_id, order_index, created_at, updated_at
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		tenantID, schoolID, p.AcademicYearID,
		p.DayOfWeek, p.PeriodName, p.StartTime, p.EndTime, p.IsBreak, p.Order,
	).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.AcademicYearID, &b.Order,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.CreateBlock: %w", ErrBlockOverlap)
		}
		return nil, fmt.Errorf("timetable.Repository.CreateBlock: %w", err)
	}
	return &b, nil
}

func (r *PgRepository) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	const query = `
		UPDATE timetable_structures
		SET day_of_week = $1, period_name = $2, start_time = $3, end_time = $4, is_break = $5,
		    academic_year_id = $6, order_index = $7, updated_at = NOW()
		WHERE id = $8 AND tenant_id = $9 AND school_id = $10
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break,
		          academic_year_id, order_index, created_at, updated_at
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		p.DayOfWeek, p.PeriodName, p.StartTime, p.EndTime, p.IsBreak,
		p.AcademicYearID, p.Order, id, tenantID, schoolID,
	).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.AcademicYearID, &b.Order,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetable.Repository.UpdateBlock: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.UpdateBlock: %w", ErrBlockOverlap)
		}
		return nil, fmt.Errorf("timetable.Repository.UpdateBlock: %w", err)
	}
	return &b, nil
}

func (r *PgRepository) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		DELETE FROM timetable_structures
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("timetable.Repository.DeleteBlock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("timetable.Repository.DeleteBlock: %w", ErrNotFound)
	}
	return nil
}

func (r *PgRepository) ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT id, tenant_id, school_id, academic_year_id, structure_id,
		       class_id, learning_area_id, teacher_id, room_identifier,
		       created_at, updated_at
		FROM cbc_timetable_slots
		WHERE tenant_id = $1 AND school_id = $2
	`)

	args := []any{f.TenantID, f.SchoolID}
	argIdx := 3

	if f.AcademicYearID != "" {
		sb.WriteString(fmt.Sprintf(" AND academic_year_id = $%d", argIdx))
		args = append(args, f.AcademicYearID)
		argIdx++
	}
	if f.StructureID != "" {
		sb.WriteString(fmt.Sprintf(" AND structure_id = $%d", argIdx))
		args = append(args, f.StructureID)
		argIdx++
	}
	if f.ClassID != "" {
		sb.WriteString(fmt.Sprintf(" AND class_id = $%d", argIdx))
		args = append(args, f.ClassID)
		argIdx++
	}
	if f.TeacherID != "" {
		sb.WriteString(fmt.Sprintf(" AND teacher_id = $%d", argIdx))
		args = append(args, f.TeacherID)
		argIdx++
	}
	if f.LearningAreaID != "" {
		sb.WriteString(fmt.Sprintf(" AND learning_area_id = $%d", argIdx))
		args = append(args, f.LearningAreaID)
		argIdx++
	}

	sb.WriteString(" ORDER BY created_at DESC")

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListSlots: %w", err)
	}
	defer rows.Close()

	var slots []Slot
	for rows.Next() {
		var s Slot
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
			&s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
			&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("timetable.Repository.ListSlots: scan: %w", err)
		}
		slots = append(slots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListSlots: rows: %w", err)
	}

	if slots == nil {
		slots = []Slot{}
	}
	return slots, nil
}

func (r *PgRepository) GetSlot(ctx context.Context, id string) (*Slot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, structure_id,
		       class_id, learning_area_id, teacher_id, room_identifier,
		       created_at, updated_at
		FROM cbc_timetable_slots
		WHERE id = $1
	`

	var s Slot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
		&s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
		&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetable.Repository.GetSlot: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("timetable.Repository.GetSlot: %w", err)
	}
	return &s, nil
}

func (r *PgRepository) CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error) {
	const query = `
		INSERT INTO cbc_timetable_slots (tenant_id, school_id, academic_year_id, structure_id,
		                                 class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, school_id, academic_year_id, structure_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	var s Slot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		tenantID, schoolID, academicYearID, p.StructureID,
		p.ClassID, p.LearningAreaID, p.TeacherID, p.RoomIdentifier,
	).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
		&s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
		&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.CreateSlot: %w", ErrConflict)
		}
		return nil, fmt.Errorf("timetable.Repository.CreateSlot: %w", err)
	}
	return &s, nil
}

func (r *PgRepository) BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, payloads []SlotPayload) ([]Slot, error) {
	if len(payloads) == 0 {
		return []Slot{}, nil
	}

	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				r.logger.Warnw("timetable.Repository.BatchCreateSlots: rollback error", "error", rbErr)
			}
		}
	}()

	var allSlots []Slot
	for _, p := range payloads {
		const query = `
			INSERT INTO cbc_timetable_slots (tenant_id, school_id, academic_year_id, structure_id,
			                                 class_id, learning_area_id, teacher_id, room_identifier)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, tenant_id, school_id, academic_year_id, structure_id,
			          class_id, learning_area_id, teacher_id, room_identifier,
			          created_at, updated_at
		`
		var s Slot
		err = tx.QueryRow(ctx, query,
			tenantID, schoolID, academicYearID, p.StructureID,
			p.ClassID, p.LearningAreaID, p.TeacherID, p.RoomIdentifier,
		).Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
			&s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
			&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: %w", ErrConflict)
			}
			return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: insert: %w", err)
		}
		allSlots = append(allSlots, s)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: commit tx: %w", err)
	}
	return allSlots, nil
}

func (r *PgRepository) UpdateSlot(ctx context.Context, id string, p UpdateSlotPayload) (*Slot, error) {
	const query = `
		UPDATE cbc_timetable_slots
		SET learning_area_id = $1, teacher_id = $2, room_identifier = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, tenant_id, school_id, academic_year_id, structure_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	var s Slot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		p.LearningAreaID, p.TeacherID, p.RoomIdentifier, id,
	).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
		&s.StructureID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
		&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", ErrConflict)
		}
		return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", err)
	}
	return &s, nil
}

func (r *PgRepository) DeleteSlot(ctx context.Context, id string) error {
	const query = `DELETE FROM cbc_timetable_slots WHERE id = $1`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("timetable.Repository.DeleteSlot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("timetable.Repository.DeleteSlot: %w", ErrNotFound)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx unique violation error code
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "unique_violation")
}
