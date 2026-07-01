package summaries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles competency summary database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// scanSummary scans a single CompetencySummary row.
func scanSummary(row pgx.Row) (*CompetencySummary, error) {
	var s CompetencySummary
	err := row.Scan(
		&s.ID, &s.TenantID, &s.StudentID, &s.LearningAreaID,
		&s.ClassID, &s.AcademicYear, &s.Term,
		&s.CalculatedLevel, &s.OverrideLevel, &s.FinalLevel,
		&s.KNECSyncStatus, &s.KNECSyncedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// scanSummariesRows scans all rows from a result set into a slice.
func scanSummariesRows(rows pgx.Rows) ([]CompetencySummary, error) {
	var summaries []CompetencySummary
	for rows.Next() {
		var s CompetencySummary
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.StudentID, &s.LearningAreaID,
			&s.ClassID, &s.AcademicYear, &s.Term,
			&s.CalculatedLevel, &s.OverrideLevel, &s.FinalLevel,
			&s.KNECSyncStatus, &s.KNECSyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("summaries.Repository.scanSummariesRows: scan: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("summaries.Repository.scanSummariesRows: rows: %w", err)
	}
	if summaries == nil {
		summaries = []CompetencySummary{}
	}
	return summaries, nil
}

// ============================================================================
// List of columns for reuse
// ============================================================================

const summaryColumns = `id, tenant_id, student_id, learning_area_id,
	class_id, academic_year, term,
	calculated_level::text, override_level::text, final_level::text,
	knec_sync_status::text, knec_synced_at::text`

// ============================================================================
// CRUD
// ============================================================================

// GetByID retrieves a single summary by primary key.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID string) (*CompetencySummary, error) {
	const query = `
		SELECT ` + summaryColumns + `
		FROM cbc_term_competency_summaries
		WHERE id = $1 AND tenant_id = $2
	`
	s, err := scanSummary(r.pool.QueryRow(ctx, query, id, tenantID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("summaries.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("summaries.Repository.GetByID: %w", err)
	}
	return s, nil
}

// List returns summaries filtered by the given query parameters.
func (r *PgRepository) List(ctx context.Context, tenantID string, query ListSummariesQuery) ([]CompetencySummary, error) {
	baseQuery := `
		SELECT ` + summaryColumns + `
		FROM cbc_term_competency_summaries
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if query.StudentID != "" {
		baseQuery += fmt.Sprintf(" AND student_id = $%d", argIdx)
		args = append(args, query.StudentID)
		argIdx++
	}
	if query.ClassID != "" {
		baseQuery += fmt.Sprintf(" AND class_id = $%d", argIdx)
		args = append(args, query.ClassID)
		argIdx++
	}
	if query.LearningAreaID != "" {
		baseQuery += fmt.Sprintf(" AND learning_area_id = $%d", argIdx)
		args = append(args, query.LearningAreaID)
		argIdx++
	}
	if query.AcademicYear > 0 {
		baseQuery += fmt.Sprintf(" AND academic_year = $%d", argIdx)
		args = append(args, query.AcademicYear)
		argIdx++
	}
	if query.Term > 0 {
		baseQuery += fmt.Sprintf(" AND term = $%d", argIdx)
		args = append(args, query.Term)
	}

	baseQuery += " ORDER BY student_id, learning_area_id, term"

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("summaries.Repository.List: %w", err)
	}
	defer rows.Close()

	return scanSummariesRows(rows)
}

// ============================================================================
// Calculation: Aggregation query
// ============================================================================

// CalculateForClass runs the modal rubric level aggregation for all students
// in a given class + term and upserts the summaries. Returns the count of
// summaries created/updated.
func (r *PgRepository) CalculateForClass(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
	// Step 1: Calculate modal rubric levels per (student, learning_area) for the given class+term.
	// Uses the mode aggregate with tie-breaking: if there are ties for mode, the highest
	// competency level wins (EE > ME > AE > BE).
	// Sub-levels are determined by the most common sub-level within the modal base level.
	const aggregationQuery = `
		WITH modal_raw AS (
			SELECT
				lrr.student_id,
				pi.sub_strand_id,
				ss.strand_id,
				ss.learning_area_id,
				lrr.rubric_level
			FROM learner_rubric_results lrr
			JOIN performance_indicators pi ON pi.id = lrr.indicator_id
			JOIN cbc_sub_strands ss ON ss.id = pi.sub_strand_id
			WHERE lrr.session_id IN (
				SELECT as2.id FROM assessment_sessions as2
				WHERE as2.class_id = $2
			)
		),
		-- Count occurrences of each base level per (student, learning_area)
		base_counts AS (
			SELECT
				student_id,
				learning_area_id,
				LEFT(rubric_level::text, 2) AS base_level,
				COUNT(*) AS cnt
			FROM modal_raw
			GROUP BY student_id, learning_area_id, LEFT(rubric_level::text, 2)
		),
		-- Rank base levels by count (desc), then by hierarchy (EE=5 > ME=4 > AE=3 > BE=2)
		ranked_base AS (
			SELECT
				student_id,
				learning_area_id,
				base_level,
				cnt,
				ROW_NUMBER() OVER (
					PARTITION BY student_id, learning_area_id
					ORDER BY cnt DESC,
						CASE base_level
							WHEN 'EE' THEN 5
							WHEN 'ME' THEN 4
							WHEN 'AE' THEN 3
							WHEN 'BE' THEN 2
							ELSE 0
						END DESC
				) AS rn
			FROM base_counts
		),
		-- Pick the winning base level per (student, learning_area)
		winning_base AS (
			SELECT student_id, learning_area_id, base_level
			FROM ranked_base
			WHERE rn = 1
		),
		-- Find the most common sub-level within the winning base level
		sub_level_mode AS (
			SELECT
				mr.student_id,
				mr.learning_area_id,
				mr.rubric_level::text AS sub_level,
				COUNT(*) AS sub_cnt
			FROM modal_raw mr
			JOIN winning_base wb ON wb.student_id = mr.student_id AND wb.learning_area_id = mr.learning_area_id
			WHERE LEFT(mr.rubric_level::text, 2) = wb.base_level
			GROUP BY mr.student_id, mr.learning_area_id, mr.rubric_level::text
		),
		ranked_sub AS (
			SELECT
				student_id,
				learning_area_id,
				sub_level,
				ROW_NUMBER() OVER (
					PARTITION BY student_id, learning_area_id
					ORDER BY sub_cnt DESC, sub_level DESC
				) AS rn
			FROM sub_level_mode
		),
		-- Final computed values
		computed AS (
			SELECT
				rs.student_id,
				rs.learning_area_id,
				rs.sub_level AS calculated_level,
				wb.base_level AS final_level
			FROM ranked_sub rs
			JOIN winning_base wb ON wb.student_id = rs.student_id AND wb.learning_area_id = rs.learning_area_id
			WHERE rs.rn = 1
		),
		-- Get class_id for each student (most recent enrollment in the given class)
		student_class AS (
			SELECT DISTINCT ON (e.student_id)
				e.student_id,
				e.class_id
			FROM cbc_student_enrollments e
			WHERE e.class_id = $2 AND e.status = 'ACTIVE'
		)
		-- Upsert into summaries table
		INSERT INTO cbc_term_competency_summaries
			(tenant_id, student_id, learning_area_id, class_id, academic_year, term,
			 calculated_level, override_level, final_level, knec_sync_status)
		SELECT
			$1::UUID AS tenant_id,
			c.student_id,
			c.learning_area_id,
			sc.class_id,
			$3 AS academic_year,
			$4 AS term,
			c.calculated_level::cbc_rubric_level_with_sub_levels,
			NULL AS override_level,
			c.final_level::cbc_rubric_level,
			'Pending'::knec_sync_status
		FROM computed c
		JOIN student_class sc ON sc.student_id = c.student_id
		WHERE sc.class_id = $2
		ON CONFLICT (student_id, learning_area_id, academic_year, term)
		DO UPDATE SET
			calculated_level = EXCLUDED.calculated_level,
			override_level   = CASE
				WHEN cbc_term_competency_summaries.override_level IS NOT NULL
				THEN cbc_term_competency_summaries.override_level
				ELSE NULL
			END,
			final_level      = CASE
				WHEN cbc_term_competency_summaries.override_level IS NOT NULL
				THEN BaseRubricLevel(cbc_term_competency_summaries.override_level::text)::cbc_rubric_level
				ELSE EXCLUDED.final_level
			END
	`

	tag, err := r.pool.Exec(ctx, aggregationQuery,
		tenantID, payload.ClassID, payload.AcademicYear, payload.Term,
	)
	if err != nil {
		return 0, fmt.Errorf("summaries.Repository.CalculateForClass: %w", err)
	}

	return int(tag.RowsAffected()), nil
}

// ============================================================================
// Override
// ============================================================================

// SetOverrideLevel updates the override_level for a summary. If overrideLevel
// is nil, clears the override. The final_level is recomputed accordingly.
func (r *PgRepository) SetOverrideLevel(ctx context.Context, id, tenantID string, overrideLevel *string) error {
	const query = `
		UPDATE cbc_term_competency_summaries
		SET
			override_level = $1::cbc_rubric_level_with_sub_levels,
			final_level = CASE
				WHEN $1 IS NOT NULL
				THEN LEFT($1, 2)::cbc_rubric_level
				ELSE LEFT(calculated_level::text, 2)::cbc_rubric_level
			END
		WHERE id = $2 AND tenant_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, overrideLevel, id, tenantID)
	if err != nil {
		return fmt.Errorf("summaries.Repository.SetOverrideLevel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("summaries.Repository.SetOverrideLevel: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// KNEC Sync
// ============================================================================

// MarkSynced updates the KNEC sync status for a summary.
func (r *PgRepository) MarkSynced(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
	const query = `
		UPDATE cbc_term_competency_summaries
		SET
			knec_sync_status = $1::knec_sync_status,
			knec_synced_at   = CASE
				WHEN $1 = 'Synced'::knec_sync_status THEN NOW()
				ELSE NULL
			END
		WHERE id = $2 AND tenant_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("summaries.Repository.MarkSynced: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("summaries.Repository.MarkSynced: %w", ErrNotFound)
	}
	return nil
}

// GetSyncStatus returns the current knec_sync_status for a summary.
func (r *PgRepository) GetSyncStatus(ctx context.Context, id, tenantID string) (string, error) {
	const query = `
		SELECT knec_sync_status::text
		FROM cbc_term_competency_summaries
		WHERE id = $1 AND tenant_id = $2
	`
	var status string
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("summaries.Repository.GetSyncStatus: %w", ErrNotFound)
		}
		return "", fmt.Errorf("summaries.Repository.GetSyncStatus: %w", err)
	}
	return status, nil
}
