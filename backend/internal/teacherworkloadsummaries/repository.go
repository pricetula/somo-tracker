package teacherworkloadsummaries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed teacher workload summaries repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

func (r *pgRepository) RefreshComputation(ctx context.Context, academicYearID string) error {
	_, err := r.pool.Exec(ctx, `SELECT fn_compute_teacher_workload_summaries($1)`, academicYearID)
	if err != nil {
		return fmt.Errorf("teacherworkloadsummaries.Repository.RefreshComputation: %w", err)
	}
	return nil
}

func (r *pgRepository) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, yearID string) (*WorkloadSummaryListResponse, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, academic_year_id,
		       total_assigned_periods, unique_subjects, classes_taught,
		       utilization_percentage, is_overcapacity, last_refreshed_at
		FROM teacher_workload_summaries
		WHERE tenant_id = $1 AND school_id = $2
		  AND user_id = $3 AND academic_year_id = $4
	`

	var s TeacherWorkloadSummary
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID, userID, yearID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.AcademicYearID,
		&s.TotalAssignedPeriods, &s.UniqueSubjects, &s.ClassesTaught,
		&s.UtilizationPercentage, &s.IsOvercapacity, &s.LastRefreshedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("teacherworkloadsummaries.Repository.ListByTeacher: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("teacherworkloadsummaries.Repository.ListByTeacher: %w", err)
	}

	return &WorkloadSummaryListResponse{
		Items: []TeacherWorkloadSummary{s},
		Total: 1,
	}, nil
}

func (r *pgRepository) ListByYear(ctx context.Context, tenantID, schoolID, yearID string) (*WorkloadSummaryListResponse, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, academic_year_id,
		       total_assigned_periods, unique_subjects, classes_taught,
		       utilization_percentage, is_overcapacity, last_refreshed_at
		FROM teacher_workload_summaries
		WHERE tenant_id = $1 AND school_id = $2
		  AND academic_year_id = $3
		ORDER BY user_id
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, yearID)
	if err != nil {
		return nil, fmt.Errorf("teacherworkloadsummaries.Repository.ListByYear: %w", err)
	}
	defer rows.Close()

	results, err := scanSummaries(rows)
	if err != nil {
		return nil, fmt.Errorf("teacherworkloadsummaries.Repository.ListByYear: %w", err)
	}

	return &WorkloadSummaryListResponse{
		Items: results,
		Total: len(results),
	}, nil
}

func (r *pgRepository) GetByTeacherYear(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, academic_year_id,
		       total_assigned_periods, unique_subjects, classes_taught,
		       utilization_percentage, is_overcapacity, last_refreshed_at
		FROM teacher_workload_summaries
		WHERE user_id = $1 AND academic_year_id = $2
	`

	var s TeacherWorkloadSummary
	err := r.pool.QueryRow(ctx, query, userID, yearID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.AcademicYearID,
		&s.TotalAssignedPeriods, &s.UniqueSubjects, &s.ClassesTaught,
		&s.UtilizationPercentage, &s.IsOvercapacity, &s.LastRefreshedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("teacherworkloadsummaries.Repository.GetByTeacherYear: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("teacherworkloadsummaries.Repository.GetByTeacherYear: %w", err)
	}

	return &s, nil
}

// scanSummaries scans rows into a slice of TeacherWorkloadSummary.
func scanSummaries(rows pgx.Rows) ([]TeacherWorkloadSummary, error) {
	var summaries []TeacherWorkloadSummary
	for rows.Next() {
		var s TeacherWorkloadSummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.AcademicYearID,
			&s.TotalAssignedPeriods, &s.UniqueSubjects, &s.ClassesTaught,
			&s.UtilizationPercentage, &s.IsOvercapacity, &s.LastRefreshedAt,
		); err != nil {
			return nil, fmt.Errorf("teacherworkloadsummaries.Repository.scanSummaries: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("teacherworkloadsummaries.Repository.scanSummaries: rows: %w", err)
	}
	if summaries == nil {
		summaries = []TeacherWorkloadSummary{}
	}
	return summaries, nil
}

var _ Repository = (*pgRepository)(nil)
