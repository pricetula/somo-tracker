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
	query := `
		SELECT b.id, b.day_of_week, b.period_name, b.start_time, b.end_time, b.is_break, b.order_index,
		       t.academic_year_id,
		       b.created_at, b.updated_at
		FROM timetable_blocks b
		JOIN timetable_tracks t ON t.tenant_id = b.tenant_id AND t.id = b.track_id
		WHERE b.tenant_id = $1 AND b.school_id = $2`
	args := []any{tenantID, schoolID}
	argIdx := 3

	if academicYearID != "" {
		query += fmt.Sprintf(" AND t.academic_year_id = $%d", argIdx)
		args = append(args, academicYearID)
		argIdx++
	}

	query += ` ORDER BY b.day_of_week ASC, b.order_index ASC, b.start_time ASC`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListBlocks: %w", err)
	}
	defer rows.Close()

	blocks := []TimeBlock{}
	for rows.Next() {
		var b TimeBlock
		if err := rows.Scan(
			&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
			&b.IsBreak, &b.OrderIndex, &b.AcademicYearID,
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
		SELECT b.id, b.day_of_week, b.period_name, b.start_time, b.end_time, b.is_break, b.order_index,
		       t.academic_year_id,
		       b.created_at, b.updated_at
		FROM timetable_blocks b
		JOIN timetable_tracks t ON t.tenant_id = b.tenant_id AND t.id = b.track_id
		WHERE b.id = $1 AND b.tenant_id = $2 AND b.school_id = $3
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.OrderIndex, &b.AcademicYearID,
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
		INSERT INTO timetable_blocks (tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break, order_index,
		          created_at, updated_at
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		tenantID, schoolID, p.TrackID,
		p.DayOfWeek, p.PeriodName, p.StartTime, p.EndTime, p.IsBreak, p.OrderIndex,
	).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.OrderIndex, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.CreateBlock: %w", ErrBlockOverlap)
		}
		return nil, fmt.Errorf("timetable.Repository.CreateBlock: %w", err)
	}
	// Fetch academic_year_id from track for the response
	b.AcademicYearID, _ = r.getAcademicYearForTrack(ctx, tenantID, p.TrackID)
	b.TrackID = p.TrackID
	return &b, nil
}

func (r *PgRepository) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	const query = `
		UPDATE timetable_blocks
		SET day_of_week = $1, period_name = $2, start_time = $3, end_time = $4, is_break = $5,
		    track_id = $6, order_index = $7, updated_at = NOW()
		WHERE id = $8 AND tenant_id = $9 AND school_id = $10
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break, order_index,
		          created_at, updated_at
	`

	var b TimeBlock
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		p.DayOfWeek, p.PeriodName, p.StartTime, p.EndTime, p.IsBreak,
		p.TrackID, p.OrderIndex, id, tenantID, schoolID,
	).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &b.StartTime, &b.EndTime,
		&b.IsBreak, &b.OrderIndex, &b.CreatedAt, &b.UpdatedAt,
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
	b.AcademicYearID, _ = r.getAcademicYearForTrack(ctx, tenantID, p.TrackID)
	b.TrackID = p.TrackID
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

func (r *PgRepository) getAcademicYearForTrack(ctx context.Context, tenantID, trackID string) (string, error) {
	var academicYearID string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx,
		`SELECT academic_year_id FROM timetable_tracks WHERE id = $1 AND tenant_id = $2`,
		trackID, tenantID,
	).Scan(&academicYearID)
	return academicYearID, err
}

func (r *PgRepository) ListAllocations(ctx context.Context, f AllocationFilter) ([]Allocation, error) {
	query := `
		SELECT 
			a.id, a.tenant_id, a.school_id, a.block_id,
			a.class_id, a.learning_area_id, a.teacher_id, a.room_identifier,
			a.created_at, a.updated_at,
			t.academic_year_id,
			c.grade_level, c.stream_id,
			la.name AS learning_area_name, la.code AS learning_area_code,
			u.full_name AS teacher_name,
			COALESCE(a.room_identifier, '') AS room_name
		FROM timetable_allocations a
		JOIN timetable_blocks b ON b.tenant_id = a.tenant_id AND b.id = a.block_id
		JOIN timetable_tracks t ON t.tenant_id = b.tenant_id AND t.id = b.track_id
		JOIN cbc_classes c ON c.tenant_id = a.tenant_id AND c.id = a.class_id
		JOIN cbc_learning_areas la ON la.tenant_id = a.tenant_id AND la.id = a.learning_area_id
		JOIN users u ON u.tenant_id = a.tenant_id AND u.id = a.teacher_id
		WHERE a.tenant_id = $1 AND a.school_id = $2
	`
	args := []any{f.TenantID, f.SchoolID}
	argIdx := 3

	if f.AcademicYearID != "" {
		query += fmt.Sprintf(" AND t.academic_year_id = $%d", argIdx)
		args = append(args, f.AcademicYearID)
		argIdx++
	}
	if f.BlockID != "" {
		query += fmt.Sprintf(" AND a.block_id = $%d", argIdx)
		args = append(args, f.BlockID)
		argIdx++
	}
	if f.ClassID != "" {
		query += fmt.Sprintf(" AND a.class_id = $%d", argIdx)
		args = append(args, f.ClassID)
		argIdx++
	}
	if f.TeacherID != "" {
		query += fmt.Sprintf(" AND a.teacher_id = $%d", argIdx)
		args = append(args, f.TeacherID)
		argIdx++
	}
	if f.LearningAreaID != "" {
		query += fmt.Sprintf(" AND a.learning_area_id = $%d", argIdx)
		args = append(args, f.LearningAreaID)
	}

	query += " ORDER BY a.created_at DESC"

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListAllocations: %w", err)
	}
	defer rows.Close()

	allocations := []Allocation{}
	for rows.Next() {
		var a Allocation
		var classGradeLevel, classStreamID string
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.SchoolID, &a.BlockID,
			&a.ClassID, &a.LearningAreaID, &a.TeacherID, &a.RoomIdentifier,
			&a.CreatedAt, &a.UpdatedAt,
			&a.AcademicYearID,
			&classGradeLevel, &classStreamID,
			&a.LearningAreaName, &a.LearningAreaCode,
			&a.TeacherName,
			&a.RoomName,
		); err != nil {
			return nil, fmt.Errorf("timetable.Repository.ListAllocations: scan: %w", err)
		}
		// Build class name from grade level and stream
		a.ClassName = fmt.Sprintf("Grade %s%s", classGradeLevel, classStreamID)
		allocations = append(allocations, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetable.Repository.ListAllocations: rows: %w", err)
	}

	return allocations, nil
}

func (r *PgRepository) GetAllocation(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error) {
	const query = `
		SELECT 
			a.id, a.tenant_id, a.school_id, a.block_id,
			a.class_id, a.learning_area_id, a.teacher_id, a.room_identifier,
			a.created_at, a.updated_at,
			t.academic_year_id,
			c.grade_level, c.stream_id,
			la.name AS learning_area_name, la.code AS learning_area_code,
			u.full_name AS teacher_name,
			COALESCE(a.room_identifier, '') AS room_name
		FROM timetable_allocations a
		JOIN timetable_blocks b ON b.tenant_id = a.tenant_id AND b.id = a.block_id
		JOIN timetable_tracks t ON t.tenant_id = b.tenant_id AND t.id = b.track_id
		JOIN cbc_classes c ON c.tenant_id = a.tenant_id AND c.id = a.class_id
		JOIN cbc_learning_areas la ON la.tenant_id = a.tenant_id AND la.id = a.learning_area_id
		JOIN users u ON u.tenant_id = a.tenant_id AND u.id = a.teacher_id
		WHERE a.id = $1 AND a.tenant_id = $2 AND a.school_id = $3
	`

	var a Allocation
	var classGradeLevel, classStreamID string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&a.ID, &a.TenantID, &a.SchoolID, &a.BlockID,
		&a.ClassID, &a.LearningAreaID, &a.TeacherID, &a.RoomIdentifier,
		&a.CreatedAt, &a.UpdatedAt,
		&a.AcademicYearID,
		&classGradeLevel, &classStreamID,
		&a.LearningAreaName, &a.LearningAreaCode,
		&a.TeacherName,
		&a.RoomName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("timetable.Repository.GetAllocation: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("timetable.Repository.GetAllocation: %w", err)
	}
	a.ClassName = fmt.Sprintf("Grade %s%s", classGradeLevel, classStreamID)
	return &a, nil
}

func (r *PgRepository) CreateAllocation(ctx context.Context, tenantID, schoolID string, p CreateAllocationPayload) (*Allocation, error) {
	const query = `
		INSERT INTO timetable_allocations (tenant_id, school_id, block_id,
		                                 class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, school_id, block_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	var a Allocation
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		tenantID, schoolID, p.BlockID,
		p.ClassID, p.LearningAreaID, p.TeacherID, p.RoomIdentifier,
	).Scan(
		&a.ID, &a.TenantID, &a.SchoolID, &a.BlockID,
		&a.ClassID, &a.LearningAreaID, &a.TeacherID,
		&a.RoomIdentifier, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.CreateAllocation: %w", classifyAllocationConflict(err))
		}
		return nil, fmt.Errorf("timetable.Repository.CreateAllocation: %w", err)
	}
	// Fetch academic_year_id from track via block for the response
	a.AcademicYearID, _ = r.getAcademicYearForBlock(ctx, tenantID, p.BlockID)
	return &a, nil
}

func (r *PgRepository) getAcademicYearForBlock(ctx context.Context, tenantID, blockID string) (string, error) {
	var academicYearID string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx,
		`SELECT t.academic_year_id FROM timetable_blocks b
		 JOIN timetable_tracks t ON t.tenant_id = b.tenant_id AND t.id = b.track_id
		 WHERE b.id = $1 AND b.tenant_id = $2`,
		blockID, tenantID,
	).Scan(&academicYearID)
	return academicYearID, err
}

func classifyAllocationConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "unique_class_slot":
			return ErrClassAllocationOccupied
		case "unique_teacher_slot":
			return ErrTeacherDoubleBooked
		case "unique_room_slot":
			return ErrRoomDoubleBooked
		}
	}
	return ErrConflict
}

func (r *PgRepository) BatchCreateAllocations(ctx context.Context, tenantID, schoolID string, payloads []CreateAllocationPayload) ([]Allocation, error) {
	if len(payloads) == 0 {
		return []Allocation{}, nil
	}

	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateAllocations: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				r.logger.Warnw("timetable.Repository.BatchCreateAllocations: rollback error", "error", rbErr)
			}
		}
	}()

	const query = `
		INSERT INTO timetable_allocations (tenant_id, school_id, block_id,
		                                 class_id, learning_area_id, teacher_id, room_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, school_id, block_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	batch := &pgx.Batch{}
	for _, p := range payloads {
		batch.Queue(query, tenantID, schoolID, p.BlockID, p.ClassID, p.LearningAreaID, p.TeacherID, p.RoomIdentifier)
	}

	results := tx.SendBatch(ctx, batch)

	allAllocations := make([]Allocation, 0, len(payloads))
	for range payloads {
		var a Allocation
		err = results.QueryRow().Scan(
			&a.ID, &a.TenantID, &a.SchoolID, &a.BlockID,
			&a.ClassID, &a.LearningAreaID, &a.TeacherID,
			&a.RoomIdentifier, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			_ = results.Close()
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("timetable.Repository.BatchCreateAllocations: %w", ErrConflict)
			}
			return nil, fmt.Errorf("timetable.Repository.BatchCreateAllocations: insert: %w", err)
		}
		// Fetch academic_year_id for each allocation
		a.AcademicYearID, _ = r.getAcademicYearForBlock(ctx, tenantID, a.BlockID)
		allAllocations = append(allAllocations, a)
	}

	// Ensure all batch results are consumed before commit
	if err = results.Close(); err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateAllocations: close batch: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("timetable.Repository.BatchCreateAllocations: commit tx: %w", err)
	}
	return allAllocations, nil
}

func (r *PgRepository) UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error) {
	const query = `
		UPDATE timetable_allocations
		SET learning_area_id = $1, teacher_id = $2, room_identifier = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5 AND school_id = $6
		RETURNING id, tenant_id, school_id, block_id,
		          class_id, learning_area_id, teacher_id, room_identifier,
		          created_at, updated_at
	`

	var a Allocation
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		p.LearningAreaID, p.TeacherID, p.RoomIdentifier, id, tenantID, schoolID,
	).Scan(
		&a.ID, &a.TenantID, &a.SchoolID, &a.BlockID,
		&a.ClassID, &a.LearningAreaID, &a.TeacherID,
		&a.RoomIdentifier, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("timetable.Repository.UpdateAllocation: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("timetable.Repository.UpdateAllocation: %w", classifyAllocationConflict(err))
		}
		return nil, fmt.Errorf("timetable.Repository.UpdateAllocation: %w", err)
	}
	// Fetch academic_year_id from track via block for the response
	a.AcademicYearID, _ = r.getAcademicYearForBlock(ctx, tenantID, a.BlockID)
	return &a, nil
}

func (r *PgRepository) DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `DELETE FROM timetable_allocations WHERE id = $1 AND tenant_id = $2 AND school_id = $3`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("timetable.Repository.DeleteAllocation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("timetable.Repository.DeleteAllocation: %w", ErrNotFound)
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
