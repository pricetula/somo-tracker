package timetable

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		FROM timetable_blocks
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3
		ORDER BY day_of_week ASC, order_index ASC, start_time ASC
	`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListBlocks: %w", err)
	}
	defer rows.Close()

	blocks := []TimeBlock{}
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

	return blocks, nil
}

func (r *PgRepository) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	const query = `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break,
		       academic_year_id, COALESCE(order_index, 0) as order_index,
		       created_at, updated_at
		FROM timetable_blocks
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.AcademicYearID, &b.Order,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("timetable.Repository.GetBlock: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("timetable.Repository.GetBlock: %w", err)
	}
	return &b, nil
}

func (r *PgRepository) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	const query = `
		INSERT INTO timetable_blocks (tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index)
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
		UPDATE timetable_blocks
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
		if errors.Is(err, pgx.ErrNoRows) {
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
		DELETE FROM timetable_blocks
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
	query := `
		SELECT id, tenant_id, school_id, academic_year_id, block_id,
		       class_id, learning_area_id, teacher_id, room_identifier,
		       created_at, updated_at
		FROM timetable_allocations
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []any{f.TenantID, f.SchoolID}
	argIdx := 3

	if f.AcademicYearID != "" {
		query += fmt.Sprintf(" AND academic_year_id = $%d", argIdx)
		args = append(args, f.AcademicYearID)
		argIdx++
	}
	if f.BlockID != "" {
		query += fmt.Sprintf(" AND block_id = $%d", argIdx)
		args = append(args, f.BlockID)
		argIdx++
	}
	if f.ClassID != "" {
		query += fmt.Sprintf(" AND class_id = $%d", argIdx)
		args = append(args, f.ClassID)
		argIdx++
	}
	if f.TeacherID != "" {
		query += fmt.Sprintf(" AND teacher_id = $%d", argIdx)
		args = append(args, f.TeacherID)
		argIdx++
	}
	if f.LearningAreaID != "" {
		query += fmt.Sprintf(" AND learning_area_id = $%d", argIdx)
		args = append(args, f.LearningAreaID)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListSlots: %w", err)
	}
	defer rows.Close()

	slots := []Slot{}
	for rows.Next() {
		var s Slot
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
			&s.BlockID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
			&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("timetable.Repository.ListSlots: scan: %w", err)
		}
		slots = append(slots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListSlots: rows: %w", err)
	}

	return slots, nil
}

func (r *PgRepository) GetSlot(ctx context.Context, id, tenantID, schoolID string) (*Slot, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_year_id, block_id,
		       class_id, learning_area_id, teacher_id, room_identifier,
		       created_at, updated_at
		FROM timetable_allocations
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	var s Slot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
		&s.BlockID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
		&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("timetable.Repository.GetSlot: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("timetable.Repository.GetSlot: %w", err)
	}
	return &s, nil
}

func (r *PgRepository) CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error) {
	const query = `
		INSERT INTO timetable_allocations (tenant_id, school_id, academic_year_id, block_id,
		                                 class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, school_id, academic_year_id, block_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	var s Slot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		tenantID, schoolID, academicYearID, p.BlockID,
		p.ClassID, p.LearningAreaID, p.TeacherID, p.RoomIdentifier,
	).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
		&s.BlockID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
		&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.CreateSlot: %w", classifySlotConflict(err))
		}
		return nil, fmt.Errorf("timetable.Repository.CreateSlot: %w", err)
	}
	return &s, nil
}

func classifySlotConflict(err error) error {
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

	const query = `
		INSERT INTO timetable_allocations (tenant_id, school_id, academic_year_id, block_id,
		                                 class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, school_id, academic_year_id, block_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	batch := &pgx.Batch{}
	for _, p := range payloads {
		batch.Queue(query, tenantID, schoolID, academicYearID, p.BlockID, p.ClassID, p.LearningAreaID, p.TeacherID, p.RoomIdentifier)
	}

	results := tx.SendBatch(ctx, batch)

	allSlots := make([]Slot, 0, len(payloads))
	for range payloads {
		var s Slot
		err = results.QueryRow().Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
			&s.BlockID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
			&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			_ = results.Close()
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: %w", ErrConflict)
			}
			return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: insert: %w", err)
		}
		allSlots = append(allSlots, s)
	}

	// Ensure all batch results are consumed before commit
	if err = results.Close(); err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: close batch: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: commit tx: %w", err)
	}
	return allSlots, nil
}

func (r *PgRepository) UpdateSlot(ctx context.Context, id, tenantID, schoolID string, p UpdateSlotPayload) (*Slot, error) {
	const query = `
		UPDATE timetable_allocations
		SET learning_area_id = $1, teacher_id = $2, room_identifier = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5 AND school_id = $6
		RETURNING id, tenant_id, school_id, academic_year_id, block_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	var s Slot
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		p.LearningAreaID, p.TeacherID, p.RoomIdentifier, id, tenantID, schoolID,
	).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.AcademicYearID,
		&s.BlockID, &s.ClassID, &s.LearningAreaID, &s.TeacherID,
		&s.RoomIdentifier, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", classifySlotConflict(err))
		}
		return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", err)
	}
	return &s, nil
}

func (r *PgRepository) DeleteSlot(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `DELETE FROM timetable_allocations WHERE id = $1 AND tenant_id = $2 AND school_id = $3`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID, schoolID)
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
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" || pgErr.Code == "23P01" // unique_violation || exclusion_violation
	}
	return false
}

func (r *PgRepository) CreateTrack(ctx context.Context, tenantID, schoolID, academicYearID, academicTermID, name, description string, isDefault bool) (*Track, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *PgRepository) UpdateTrack(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *PgRepository) DeleteTrack(ctx context.Context, id, tenantID, schoolID string) error {
	return fmt.Errorf("not implemented")
}
func (r *PgRepository) CreateAllocation(ctx context.Context, tenantID, schoolID, blockID string, p CreateAllocationPayload) (*Allocation, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *PgRepository) UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *PgRepository) DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error {
	return fmt.Errorf("not implemented")
}
