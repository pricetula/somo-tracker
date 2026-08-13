package cohortpositions

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed cohort positions repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

// RefreshTerm calls the PL/pgSQL batch function to recompute cohort positions
// for all students in the given academic term.
func (r *pgRepository) RefreshTerm(ctx context.Context, termID string) error {
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, `SELECT fn_compute_cohort_positions_for_term($1)`, termID)
	if err != nil {
		return fmt.Errorf("cohortpositions.Repository.RefreshTerm: %w", err)
	}
	return nil
}

// GetByStudentTerm returns the cohort position for a specific student in a term.
func (r *pgRepository) GetByStudentTerm(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error) {
	var s CohortPositionSummary
	var lastRefreshedAt, createdAt, updatedAt time.Time

	err := database.FromContext(ctx, r.pool).QueryRow(ctx, `
		SELECT
			id, tenant_id, school_id, student_id, class_id, academic_term_id,
			student_score,
			class_rank, class_headcount, class_average, class_percentile,
			grade_rank, grade_headcount, grade_average, grade_percentile,
			variance_from_grade_mean,
			last_refreshed_at, created_at, updated_at
		FROM student_cohort_position_summaries
		WHERE student_id = $1 AND academic_term_id = $2 AND tenant_id = $3
	`, studentID, termID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.ClassID, &s.AcademicTermID,
		&s.StudentScore,
		&s.ClassRank, &s.ClassHeadcount, &s.ClassAverage, &s.ClassPercentile,
		&s.GradeRank, &s.GradeHeadcount, &s.GradeAverage, &s.GradePercentile,
		&s.VarianceFromGradeMean,
		&lastRefreshedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cohortpositions.Repository.GetByStudentTerm: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cohortpositions.Repository.GetByStudentTerm: %w", err)
	}

	s.LastRefreshedAt = lastRefreshedAt
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt

	return &s, nil
}

// ListByClassTerm returns all cohort positions for a class in a term.
func (r *pgRepository) ListByClassTerm(ctx context.Context, classID, termID, tenantID string) ([]CohortPositionSummary, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, `
		SELECT
			id, tenant_id, school_id, student_id, class_id, academic_term_id,
			student_score,
			class_rank, class_headcount, class_average, class_percentile,
			grade_rank, grade_headcount, grade_average, grade_percentile,
			variance_from_grade_mean,
			last_refreshed_at, created_at, updated_at
		FROM student_cohort_position_summaries
		WHERE class_id = $1 AND academic_term_id = $2 AND tenant_id = $3
		ORDER BY class_rank ASC NULLS LAST
	`, classID, termID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("cohortpositions.Repository.ListByClassTerm: %w", err)
	}
	defer rows.Close()

	var items []CohortPositionSummary
	for rows.Next() {
		var s CohortPositionSummary
		var lastRefreshedAt, createdAt, updatedAt time.Time
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.ClassID, &s.AcademicTermID,
			&s.StudentScore,
			&s.ClassRank, &s.ClassHeadcount, &s.ClassAverage, &s.ClassPercentile,
			&s.GradeRank, &s.GradeHeadcount, &s.GradeAverage, &s.GradePercentile,
			&s.VarianceFromGradeMean,
			&lastRefreshedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("cohortpositions.Repository.ListByClassTerm: scan: %w", err)
		}
		s.LastRefreshedAt = lastRefreshedAt
		s.CreatedAt = createdAt
		s.UpdatedAt = updatedAt
		items = append(items, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cohortpositions.Repository.ListByClassTerm: rows: %w", err)
	}

	return items, nil
}

// ListByGradeTerm returns all cohort positions for all classes at the same
// grade level in a term (across the school). Resolves the grade level by
// joining through cbc_classes.
func (r *pgRepository) ListByGradeTerm(ctx context.Context, schoolID, gradeLevel, termID, tenantID string) ([]CohortPositionSummary, error) {
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, `
		SELECT
			cps.id, cps.tenant_id, cps.school_id, cps.student_id, cps.class_id, cps.academic_term_id,
			cps.student_score,
			cps.class_rank, cps.class_headcount, cps.class_average, cps.class_percentile,
			cps.grade_rank, cps.grade_headcount, cps.grade_average, cps.grade_percentile,
			cps.variance_from_grade_mean,
			cps.last_refreshed_at, cps.created_at, cps.updated_at
		FROM student_cohort_position_summaries cps
		JOIN cbc_classes cc ON cc.id = cps.class_id AND cc.tenant_id = cps.tenant_id
		WHERE cc.grade_level::TEXT = $1
		  AND cps.academic_term_id = $2
		  AND cps.school_id = $3
		  AND cps.tenant_id = $4
		ORDER BY cps.grade_rank ASC NULLS LAST
	`, gradeLevel, termID, schoolID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("cohortpositions.Repository.ListByGradeTerm: %w", err)
	}
	defer rows.Close()

	var items []CohortPositionSummary
	for rows.Next() {
		var s CohortPositionSummary
		var lastRefreshedAt, createdAt, updatedAt time.Time
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.StudentID, &s.ClassID, &s.AcademicTermID,
			&s.StudentScore,
			&s.ClassRank, &s.ClassHeadcount, &s.ClassAverage, &s.ClassPercentile,
			&s.GradeRank, &s.GradeHeadcount, &s.GradeAverage, &s.GradePercentile,
			&s.VarianceFromGradeMean,
			&lastRefreshedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("cohortpositions.Repository.ListByGradeTerm: scan: %w", err)
		}
		s.LastRefreshedAt = lastRefreshedAt
		s.CreatedAt = createdAt
		s.UpdatedAt = updatedAt
		items = append(items, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cohortpositions.Repository.ListByGradeTerm: rows: %w", err)
	}

	return items, nil
}
