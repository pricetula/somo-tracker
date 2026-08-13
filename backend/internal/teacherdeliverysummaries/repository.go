package teacherdeliverysummaries

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

// NewRepository creates a new PostgreSQL-backed teacher delivery summaries repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

func (r *pgRepository) RefreshComputation(ctx context.Context, termID string) error {
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, `SELECT fn_compute_teacher_delivery_summaries($1)`, termID)
	if err != nil {
		return fmt.Errorf("teacherdeliverysummaries.Repository.RefreshComputation: %w", err)
	}
	return nil
}

func (r *pgRepository) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string) (*DeliverySummaryListResponse, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, academic_term_id,
		       total_assigned_slots, marked_slots, missed_slots,
		       sessions_created, sessions_approved,
		       on_time_submission_rate, last_refreshed_at::TEXT
		FROM teacher_delivery_summaries
		WHERE tenant_id = $1 AND school_id = $2
		  AND user_id = $3 AND academic_term_id = $4
	`

	var s TeacherDeliverySummary
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID, schoolID, userID, termID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.AcademicTermID,
		&s.TotalAssignedSlots, &s.MarkedSlots, &s.MissedSlots,
		&s.SessionsCreated, &s.SessionsApproved,
		&s.OnTimeSubmissionRate, &s.LastRefreshedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("teacherdeliverysummaries.Repository.ListByTeacher: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("teacherdeliverysummaries.Repository.ListByTeacher: %w", err)
	}

	return &DeliverySummaryListResponse{
		Items: []TeacherDeliverySummary{s},
		Total: 1,
	}, nil
}

func (r *pgRepository) ListByTerm(ctx context.Context, tenantID, schoolID, termID string) (*DeliverySummaryListResponse, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, academic_term_id,
		       total_assigned_slots, marked_slots, missed_slots,
		       sessions_created, sessions_approved,
		       on_time_submission_rate, last_refreshed_at::TEXT
		FROM teacher_delivery_summaries
		WHERE tenant_id = $1 AND school_id = $2
		  AND academic_term_id = $3
		ORDER BY user_id
	`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("teacherdeliverysummaries.Repository.ListByTerm: %w", err)
	}
	defer rows.Close()

	results, err := scanSummaries(rows)
	if err != nil {
		return nil, fmt.Errorf("teacherdeliverysummaries.Repository.ListByTerm: %w", err)
	}

	return &DeliverySummaryListResponse{
		Items: results,
		Total: len(results),
	}, nil
}

func (r *pgRepository) GetByTeacherTerm(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error) {
	const query = `
		SELECT id, tenant_id, school_id, user_id, academic_term_id,
		       total_assigned_slots, marked_slots, missed_slots,
		       sessions_created, sessions_approved,
		       on_time_submission_rate, last_refreshed_at::TEXT
		FROM teacher_delivery_summaries
		WHERE user_id = $1 AND academic_term_id = $2
	`

	var s TeacherDeliverySummary
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, userID, termID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.AcademicTermID,
		&s.TotalAssignedSlots, &s.MarkedSlots, &s.MissedSlots,
		&s.SessionsCreated, &s.SessionsApproved,
		&s.OnTimeSubmissionRate, &s.LastRefreshedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("teacherdeliverysummaries.Repository.GetByTeacherTerm: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("teacherdeliverysummaries.Repository.GetByTeacherTerm: %w", err)
	}

	return &s, nil
}

// scanSummaries scans rows into a slice of TeacherDeliverySummary.
func scanSummaries(rows pgx.Rows) ([]TeacherDeliverySummary, error) {
	var summaries []TeacherDeliverySummary
	for rows.Next() {
		var s TeacherDeliverySummary
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.UserID, &s.AcademicTermID,
			&s.TotalAssignedSlots, &s.MarkedSlots, &s.MissedSlots,
			&s.SessionsCreated, &s.SessionsApproved,
			&s.OnTimeSubmissionRate, &s.LastRefreshedAt,
		); err != nil {
			return nil, fmt.Errorf("teacherdeliverysummaries.Repository.scanSummaries: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("teacherdeliverysummaries.Repository.scanSummaries: rows: %w", err)
	}
	if summaries == nil {
		summaries = []TeacherDeliverySummary{}
	}
	return summaries, nil
}

var _ Repository = (*pgRepository)(nil)
