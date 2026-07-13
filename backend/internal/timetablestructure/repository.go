package timetablestructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles time block database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ListByDay returns all time blocks for a tenant, school, academic year, and day of week.
func (r *PgRepository) ListByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) ([]TimeBlock, error) {
	const query = `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
		FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3 AND day_of_week = $4
		ORDER BY start_time ASC
	`

	return r.queryBlocks(ctx, query, tenantID, schoolID, academicYearID, dayOfWeek)
}

// ListAll returns all time blocks for a tenant, school, and academic year.
func (r *PgRepository) ListAll(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error) {
	const query = `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
		FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3
		ORDER BY day_of_week ASC, start_time ASC
	`

	return r.queryBlocks(ctx, query, tenantID, schoolID, academicYearID)
}

func (r *PgRepository) queryBlocks(ctx context.Context, query string, args ...interface{}) ([]TimeBlock, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.queryBlocks: %w", err)
	}
	defer rows.Close()

	var blocks []TimeBlock
	for rows.Next() {
		var b TimeBlock
		var startTime, endTime time.Time
		if err := rows.Scan(&b.ID, &b.DayOfWeek, &b.PeriodName, &startTime, &endTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("timetablestructure.Repository.queryBlocks: scan: %w", err)
		}
		b.StartTime = startTime.Format("15:04")
		b.EndTime = endTime.Format("15:04")
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.queryBlocks: rows: %w", err)
	}

	if blocks == nil {
		blocks = []TimeBlock{}
	}

	return blocks, nil
}

// GetByID retrieves a time block by ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	const query = `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
		FROM timetable_structures
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	var b TimeBlock
	var startTime, endTime time.Time
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &startTime, &endTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetablestructure.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("timetablestructure.Repository.GetByID: %w", err)
	}
	b.StartTime = startTime.Format("15:04")
	b.EndTime = endTime.Format("15:04")
	return &b, nil
}

// Create inserts a new time block and returns it.
func (r *PgRepository) Create(ctx context.Context, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
	overlap, err := r.FindOverlappingBlock(ctx, tenantID, schoolID, block.DayOfWeek, block.StartTime, block.EndTime, "")
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.Create: %w", err)
	}
	if overlap != nil {
		return nil, fmt.Errorf(
			"timetablestructure.Repository.Create: time block collides with existing %q block (%s - %s): %w",
			overlap.PeriodName, overlap.StartTime, overlap.EndTime, ErrBlockOverlap,
		)
	}

	const query = `
		INSERT INTO timetable_structures (tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break)
		VALUES ($1, $2, $3::UUID, $4, $5, $6::TIME, $7::TIME, $8)
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
	`

	var b TimeBlock
	var startTime, endTime time.Time
	err = r.pool.QueryRow(ctx, query,
		tenantID, schoolID, block.AcademicYearID, block.DayOfWeek, block.PeriodName, block.StartTime, block.EndTime, block.IsBreak,
	).Scan(&b.ID, &b.DayOfWeek, &b.PeriodName, &startTime, &endTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if isOverlapViolation(err) {
			return nil, fmt.Errorf("timetablestructure.Repository.Create: %w", ErrBlockOverlap)
		}
		return nil, fmt.Errorf("timetablestructure.Repository.Create: %w", err)
	}
	b.StartTime = startTime.Format("15:04")
	b.EndTime = endTime.Format("15:04")
	return &b, nil
}

// BatchCreate inserts multiple time blocks atomically within a single transaction.
func (r *PgRepository) BatchCreate(ctx context.Context, tenantID, schoolID string, blocks []CreateTimeBlockPayload) ([]TimeBlock, error) {
	if len(blocks) == 0 {
		return []TimeBlock{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.BatchCreate: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		INSERT INTO timetable_structures (tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break)
		VALUES ($1, $2, $3::UUID, $4, $5, $6::TIME, $7::TIME, $8)
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
	`

	// Pre-check: validate each block against existing data within the transaction.
	for _, b := range blocks {
		if overlaps, err := r.findOverlappingInTx(ctx, tx, tenantID, schoolID, b.DayOfWeek, b.StartTime, b.EndTime, ""); err != nil {
			return nil, fmt.Errorf("timetablestructure.Repository.BatchCreate: %w", err)
		} else if overlaps != nil {
			return nil, fmt.Errorf(
				"timetablestructure.Repository.BatchCreate: block (%s - %s) collides with existing %q block (%s - %s): %w",
				b.StartTime, b.EndTime, overlaps.PeriodName, overlaps.StartTime, overlaps.EndTime, ErrBlockOverlap,
			)
		}
	}

	var results []TimeBlock
	for _, block := range blocks {
		var b TimeBlock
		var startTime, endTime time.Time
		err := tx.QueryRow(ctx, query,
			tenantID, schoolID, block.AcademicYearID, block.DayOfWeek, block.PeriodName, block.StartTime, block.EndTime, block.IsBreak,
		).Scan(&b.ID, &b.DayOfWeek, &b.PeriodName, &startTime, &endTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("timetablestructure.Repository.BatchCreate: %w", err)
		}
		b.StartTime = startTime.Format("15:04")
		b.EndTime = endTime.Format("15:04")
		results = append(results, b)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.BatchCreate: commit: %w", err)
	}

	return results, nil
}

// ReplicateDay copies all blocks from sourceDay to each targetDay within the
// same academic year. This is the "Mass Replication" ROI feature — admins
// define one day and replicate to others in a single batch.
func (r *PgRepository) ReplicateDay(ctx context.Context, tenantID, schoolID string, sourceDay int, targetDays []int) ([]TimeBlock, error) {
	if len(targetDays) == 0 {
		return []TimeBlock{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// First, get the source blocks to derive academic_year_id
	const sourceQuery = `
		SELECT DISTINCT academic_year_id
		FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND day_of_week = $3
	`
	var academicYearID string
	if err := tx.QueryRow(ctx, sourceQuery, tenantID, schoolID, sourceDay).Scan(&academicYearID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: no blocks found for source day %d: %w", sourceDay, ErrNotFound)
		}
		return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: %w", err)
	}

	// Get all source blocks ordered by start_time
	const selectQuery = `
		SELECT period_name, start_time, end_time, is_break
		FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND day_of_week = $3
		ORDER BY start_time ASC
	`
	rows, err := tx.Query(ctx, selectQuery, tenantID, schoolID, sourceDay)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: %w", err)
	}
	defer rows.Close()

	type sourceBlock struct {
		PeriodName string
		StartTime  time.Time
		EndTime    time.Time
		IsBreak    bool
	}

	var sourceBlocks []sourceBlock
	for rows.Next() {
		var sb sourceBlock
		if err := rows.Scan(&sb.PeriodName, &sb.StartTime, &sb.EndTime, &sb.IsBreak); err != nil {
			return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: scan: %w", err)
		}
		sourceBlocks = append(sourceBlocks, sb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: rows: %w", err)
	}

	if len(sourceBlocks) == 0 {
		return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: %w", ErrNotFound)
	}

	// For each target day, delete existing blocks then insert copies of source blocks
	var results []TimeBlock
	const insertQuery = `
		INSERT INTO timetable_structures (tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break)
		VALUES ($1, $2, $3::UUID, $4, $5, $6::TIME, $7::TIME, $8)
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
	`
	const deleteQuery = `
		DELETE FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3 AND day_of_week = $4
	`

	for _, targetDay := range targetDays {
		if targetDay == sourceDay {
			continue
		}

		// Delete all existing blocks for the target day within the tx
		_, err := tx.Exec(ctx, deleteQuery, tenantID, schoolID, academicYearID, targetDay)
		if err != nil {
			return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: delete day %d: %w", targetDay, err)
		}

		for _, sb := range sourceBlocks {
			var b TimeBlock
			var st, et time.Time
			err := tx.QueryRow(ctx, insertQuery,
				tenantID, schoolID, academicYearID, targetDay, sb.PeriodName, sb.StartTime, sb.EndTime, sb.IsBreak,
			).Scan(&b.ID, &b.DayOfWeek, &b.PeriodName, &st, &et, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt)
			if err != nil {
				return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: insert day %d: %w", targetDay, err)
			}
			b.StartTime = st.Format("15:04")
			b.EndTime = et.Format("15:04")
			results = append(results, b)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.ReplicateDay: commit: %w", err)
	}

	return results, nil
}

// Update modifies a time block's properties and returns the updated record.
func (r *PgRepository) Update(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
	overlap, err := r.FindOverlappingBlock(ctx, tenantID, schoolID, block.DayOfWeek, block.StartTime, block.EndTime, id)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Repository.Update: %w", err)
	}
	if overlap != nil {
		return nil, fmt.Errorf(
			"timetablestructure.Repository.Update: time block collides with existing %q block (%s - %s): %w",
			overlap.PeriodName, overlap.StartTime, overlap.EndTime, ErrBlockOverlap,
		)
	}

	const query = `
		UPDATE timetable_structures
		SET day_of_week = $1, period_name = $2, start_time = $3::TIME, end_time = $4::TIME, is_break = $5, updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7 AND school_id = $8
		RETURNING id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
	`

	var b TimeBlock
	var startTime, endTime time.Time
	err = r.pool.QueryRow(ctx, query,
		block.DayOfWeek, block.PeriodName, block.StartTime, block.EndTime, block.IsBreak, id, tenantID, schoolID,
	).Scan(&b.ID, &b.DayOfWeek, &b.PeriodName, &startTime, &endTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("timetablestructure.Repository.Update: %w", ErrNotFound)
		}
		if isOverlapViolation(err) {
			return nil, fmt.Errorf("timetablestructure.Repository.Update: %w", ErrBlockOverlap)
		}
		return nil, fmt.Errorf("timetablestructure.Repository.Update: %w", err)
	}
	b.StartTime = startTime.Format("15:04")
	b.EndTime = endTime.Format("15:04")
	return &b, nil
}

// HasLinkedLessons checks whether any cbc_timetable_slots reference this
// structural time block.
func (r *PgRepository) HasLinkedLessons(ctx context.Context, id, tenantID, schoolID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM cbc_timetable_slots
		WHERE structure_id = $1
	`

	var count int
	err := r.pool.QueryRow(ctx, query, id).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("timetablestructure.Repository.HasLinkedLessons: %w", err)
	}
	return count, nil
}

// DeleteByDay removes all time blocks for a given day within an academic year.
func (r *PgRepository) DeleteByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) error {
	const query = `
		DELETE FROM timetable_structures
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3 AND day_of_week = $4
	`
	tag, err := r.pool.Exec(ctx, query, tenantID, schoolID, academicYearID, dayOfWeek)
	if err != nil {
		return fmt.Errorf("timetablestructure.Repository.DeleteByDay: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("timetablestructure.Repository.DeleteByDay: %w", ErrNotFound)
	}
	return nil
}

// Delete removes a time block by ID. Returns ErrBlockHasLessons if the block
// is linked to live scheduled lessons.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	count, err := r.HasLinkedLessons(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("timetablestructure.Repository.Delete: %w", err)
	}
	if count > 0 {
		return fmt.Errorf(
			"timetablestructure.Repository.Delete: %w (linked to %d scheduled lessons)",
			ErrBlockHasLessons, count,
		)
	}

	// Check if the block also has linked cbc_timetable_slots via structure_id (ON DELETE CASCADE will handle)
	const query = `
		DELETE FROM timetable_structures
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("timetablestructure.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("timetablestructure.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// FindOverlappingBlock returns the first block that overlaps with the given
// time range on the same day, excluding the block with the given ID.
func (r *PgRepository) FindOverlappingBlock(ctx context.Context, tenantID, schoolID string, dayOfWeek int, startTime, endTime string, excludeID string) (*TimeBlock, error) {
	query := `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
		FROM timetable_structures
		WHERE tenant_id = $1
		  AND school_id = $2
		  AND day_of_week = $3
		  AND start_time < $5::TIME
		  AND end_time > $4::TIME
	`
	args := []interface{}{tenantID, schoolID, dayOfWeek, startTime, endTime}

	if excludeID != "" {
		query += ` AND id <> $6`
		args = append(args, excludeID)
	}
	query += ` ORDER BY start_time ASC LIMIT 1`

	var b TimeBlock
	var sTime, eTime time.Time
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &sTime, &eTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("timetablestructure.Repository.FindOverlappingBlock: %w", err)
	}
	b.StartTime = sTime.Format("15:04")
	b.EndTime = eTime.Format("15:04")
	return &b, nil
}

// findOverlappingInTx is the same as FindOverlappingBlock but runs within a
// given transaction. Used for batch validation.
func (r *PgRepository) findOverlappingInTx(ctx context.Context, tx pgx.Tx, tenantID, schoolID string, dayOfWeek int, startTime, endTime string, excludeID string) (*TimeBlock, error) {
	query := `
		SELECT id, day_of_week, period_name, start_time, end_time, is_break, academic_year_id, created_at, updated_at
		FROM timetable_structures
		WHERE tenant_id = $1
		  AND school_id = $2
		  AND day_of_week = $3
		  AND start_time < $5::TIME
		  AND end_time > $4::TIME
	`
	args := []interface{}{tenantID, schoolID, dayOfWeek, startTime, endTime}

	if excludeID != "" {
		query += ` AND id <> $6`
		args = append(args, excludeID)
	}
	query += ` ORDER BY start_time ASC LIMIT 1`

	var b TimeBlock
	var sTime, eTime time.Time
	err := tx.QueryRow(ctx, query, args...).Scan(
		&b.ID, &b.DayOfWeek, &b.PeriodName, &sTime, &eTime, &b.IsBreak, &b.AcademicYearID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("timetablestructure.Repository.findOverlappingInTx: %w", err)
	}
	b.StartTime = sTime.Format("15:04")
	b.EndTime = eTime.Format("15:04")
	return &b, nil
}

// isOverlapViolation checks if an error is the GiST exclusion constraint violation.
func isOverlapViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23P01"
	}
	return false
}

// compile-time interface check
var _ Repository = (*PgRepository)(nil)
