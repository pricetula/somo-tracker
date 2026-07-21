package teacherperformance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository defines the contract for teacher performance persistence.
type Repository interface {
	// RefreshComputation triggers the batch computation of teacher performance
	// summaries for all SUBJECT_TEACHER assignments in the given term.
	RefreshComputation(ctx context.Context, termID string) error

	// ListByTeacher returns performance summaries for a specific teacher in a
	// given term, optionally filtered by learning area.
	ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error)

	// ListByTerm returns all teacher performance summaries for a given term,
	// optionally filtered by class or learning area.
	ListByTerm(ctx context.Context, tenantID, schoolID, termID string, classID, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error)

	// GetByTeacherClassSubject returns a single summary row by its grain.
	GetByTeacherClassSubject(ctx context.Context, userID, learningAreaID, classID, termID string) (*TeacherSubjectPerformanceSummary, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed teacher performance repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

func (r *pgRepository) RefreshComputation(ctx context.Context, termID string) error {
	_, err := r.pool.Exec(ctx, `SELECT fn_compute_teacher_subject_performance_summaries($1)`, termID)
	if err != nil {
		return fmt.Errorf("teacherperformance.Repository.RefreshComputation: %w", err)
	}
	return nil
}

func (r *pgRepository) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error) {
	var rows pgx.Rows
	var err error

	if learningAreaID != nil && *learningAreaID != "" {
		const query = `
			SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
			       academic_term_id, subject_mean_score, cohort_mastery_rate,
			       student_growth_rate, assessment_timeliness_index,
			       strand_coverage_rate, last_refreshed_at
			FROM teacher_subject_performance_summaries
			WHERE tenant_id = $1 AND school_id = $2
			  AND user_id = $3 AND academic_term_id = $4
			  AND learning_area_id = $5
		`
		rows, err = r.pool.Query(ctx, query, tenantID, schoolID, userID, termID, *learningAreaID)
	} else {
		const query = `
			SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
			       academic_term_id, subject_mean_score, cohort_mastery_rate,
			       student_growth_rate, assessment_timeliness_index,
			       strand_coverage_rate, last_refreshed_at
			FROM teacher_subject_performance_summaries
			WHERE tenant_id = $1 AND school_id = $2
			  AND user_id = $3 AND academic_term_id = $4
			ORDER BY subject_mean_score DESC NULLS LAST
		`
		rows, err = r.pool.Query(ctx, query, tenantID, schoolID, userID, termID)
	}
	if err != nil {
		return nil, fmt.Errorf("teacherperformance.Repository.ListByTeacher: %w", err)
	}
	defer rows.Close()

	return scanSummaries(rows)
}

func (r *pgRepository) ListByTerm(ctx context.Context, tenantID, schoolID, termID string, classID, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error) {
	var rows pgx.Rows
	var err error

	switch {
	case classID != nil && *classID != "" && learningAreaID != nil && *learningAreaID != "":
		const query = `
			SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
			       academic_term_id, subject_mean_score, cohort_mastery_rate,
			       student_growth_rate, assessment_timeliness_index,
			       strand_coverage_rate, last_refreshed_at
			FROM teacher_subject_performance_summaries
			WHERE tenant_id = $1 AND school_id = $2
			  AND academic_term_id = $3 AND class_id = $4
			  AND learning_area_id = $5
		`
		rows, err = r.pool.Query(ctx, query, tenantID, schoolID, termID, *classID, *learningAreaID)
	case classID != nil && *classID != "":
		const query = `
			SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
			       academic_term_id, subject_mean_score, cohort_mastery_rate,
			       student_growth_rate, assessment_timeliness_index,
			       strand_coverage_rate, last_refreshed_at
			FROM teacher_subject_performance_summaries
			WHERE tenant_id = $1 AND school_id = $2
			  AND academic_term_id = $3 AND class_id = $4
			ORDER BY subject_mean_score DESC NULLS LAST
		`
		rows, err = r.pool.Query(ctx, query, tenantID, schoolID, termID, *classID)
	case learningAreaID != nil && *learningAreaID != "":
		const query = `
			SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
			       academic_term_id, subject_mean_score, cohort_mastery_rate,
			       student_growth_rate, assessment_timeliness_index,
			       strand_coverage_rate, last_refreshed_at
			FROM teacher_subject_performance_summaries
			WHERE tenant_id = $1 AND school_id = $2
			  AND academic_term_id = $3 AND learning_area_id = $4
			ORDER BY subject_mean_score DESC NULLS LAST
		`
		rows, err = r.pool.Query(ctx, query, tenantID, schoolID, termID, *learningAreaID)
	default:
		const query = `
			SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
			       academic_term_id, subject_mean_score, cohort_mastery_rate,
			       student_growth_rate, assessment_timeliness_index,
			       strand_coverage_rate, last_refreshed_at
			FROM teacher_subject_performance_summaries
			WHERE tenant_id = $1 AND school_id = $2
			  AND academic_term_id = $3
			ORDER BY subject_mean_score DESC NULLS LAST
		`
		rows, err = r.pool.Query(ctx, query, tenantID, schoolID, termID)
	}
	if err != nil {
		return nil, fmt.Errorf("teacherperformance.Repository.ListByTerm: %w", err)
	}
	defer rows.Close()

	return scanSummaries(rows)
}

func (r *pgRepository) GetByTeacherClassSubject(ctx context.Context, userID, learningAreaID, classID, termID string) (*TeacherSubjectPerformanceSummary, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, learning_area_id, class_id,
		       academic_term_id, subject_mean_score, cohort_mastery_rate,
		       student_growth_rate, assessment_timeliness_index,
		       strand_coverage_rate, last_refreshed_at
		FROM teacher_subject_performance_summaries
		WHERE user_id = $1 AND learning_area_id = $2
		  AND class_id = $3 AND academic_term_id = $4
	`
	var s TeacherSubjectPerformanceSummary
	err := r.pool.QueryRow(ctx, query, userID, learningAreaID, classID, termID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.LearningAreaID, &s.ClassID,
		&s.AcademicTermID, &s.SubjectMeanScore, &s.CohortMasteryRate,
		&s.StudentGrowthRate, &s.AssessmentTimelinessIdx,
		&s.StrandCoverageRate, &s.LastRefreshedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("teacherperformance.Repository.GetByTeacherClassSubject: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("teacherperformance.Repository.GetByTeacherClassSubject: %w", err)
	}
	return &s, nil
}

// scanSummaries scans rows into a slice of TeacherSubjectPerformanceSummary.
func scanSummaries(rows pgx.Rows) ([]TeacherSubjectPerformanceSummary, error) {
	var summaries []TeacherSubjectPerformanceSummary
	for rows.Next() {
		var s TeacherSubjectPerformanceSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.LearningAreaID, &s.ClassID,
			&s.AcademicTermID, &s.SubjectMeanScore, &s.CohortMasteryRate,
			&s.StudentGrowthRate, &s.AssessmentTimelinessIdx,
			&s.StrandCoverageRate, &s.LastRefreshedAt,
		); err != nil {
			return nil, fmt.Errorf("teacherperformance.Repository.scanSummaries: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("teacherperformance.Repository.scanSummaries: rows: %w", err)
	}
	if summaries == nil {
		summaries = []TeacherSubjectPerformanceSummary{}
	}
	return summaries, nil
}

var _ Repository = (*pgRepository)(nil)
