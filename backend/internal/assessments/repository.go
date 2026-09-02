package assessments

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

// PgRepository handles assessment session database operations.
type PgRepository struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools, logger *zap.SugaredLogger) *PgRepository {
	return &PgRepository{pool: pools.PG, logger: logger}
}

// List returns a paginated list of assessment sessions.
func (r *PgRepository) List(ctx context.Context, filter SessionListFilter) (*SessionListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	baseSelect := `
		SELECT id, tenant_id, school_id, class_id, learning_area_id,
		       academic_term_id, academic_year_id, name, evaluation_method::text,
		       max_points, grading_scale_profile_id, status::text,
		       rejection_comment, submitted_by, approved_by, scheduled_date,
		       created_at, updated_at, created_by
		FROM assessment_sessions
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []interface{}{filter.TenantID, filter.SchoolID}
	argIdx := 3

	if filter.ClassID != "" {
		baseSelect += fmt.Sprintf(" AND class_id = $%d", argIdx)
		args = append(args, filter.ClassID)
		argIdx++
	}
	if filter.LearningAreaID != "" {
		baseSelect += fmt.Sprintf(" AND learning_area_id = $%d", argIdx)
		args = append(args, filter.LearningAreaID)
		argIdx++
	}
	if filter.AcademicTermID != "" {
		baseSelect += fmt.Sprintf(" AND academic_term_id = $%d", argIdx)
		args = append(args, filter.AcademicTermID)
		argIdx++
	}
	if filter.Status != "" {
		baseSelect += fmt.Sprintf(" AND status::text = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.EvaluationMethod != "" {
		baseSelect += fmt.Sprintf(" AND evaluation_method::text = $%d", argIdx)
		args = append(args, filter.EvaluationMethod)
		argIdx++
	}

	// Count
	countQuery := "SELECT COUNT(*) FROM (" + baseSelect + ") AS sub"
	var total int
	if err := database.FromContext(ctx, r.pool).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("assessments.Repository.List: count: %w", err)
	}

	// Data
	dataQuery := baseSelect + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	dataArgs := append(args, filter.Limit, offset)

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.List: query: %w", err)
	}
	defer rows.Close()

	var items []AssessmentSession
	for rows.Next() {
		var s AssessmentSession
		var evaluationMethod, status string
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.LearningAreaID,
			&s.AcademicTermID, &s.AcademicYearID, &s.Name, &evaluationMethod,
			&s.MaxPoints, &s.GradingScaleProfileID, &status,
			&s.RejectionComment, &s.SubmittedBy, &s.ApprovedBy, &s.ScheduledDate,
			&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("assessments.Repository.List: scan: %w", err)
		}
		s.EvaluationMethod = evaluationMethod
		s.Status = status
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessments.Repository.List: rows: %w", err)
	}
	if items == nil {
		items = []AssessmentSession{}
	}

	return &SessionListResult{
		Items: items,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

// GetByID retrieves a session by ID scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error) {
	const query = `
		SELECT id, tenant_id, school_id, class_id, learning_area_id,
		       academic_term_id, academic_year_id, name, evaluation_method::text,
		       max_points, grading_scale_profile_id, status::text,
		       rejection_comment, submitted_by, approved_by, scheduled_date,
		       created_at, updated_at, created_by
		FROM assessment_sessions
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	var s AssessmentSession
	var evaluationMethod, status string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.LearningAreaID,
		&s.AcademicTermID, &s.AcademicYearID, &s.Name, &evaluationMethod,
		&s.MaxPoints, &s.GradingScaleProfileID, &status,
		&s.RejectionComment, &s.SubmittedBy, &s.ApprovedBy, &s.ScheduledDate,
		&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("assessments.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("assessments.Repository.GetByID: %w", err)
	}
	s.EvaluationMethod = evaluationMethod
	s.Status = status
	return &s, nil
}

// Create inserts a new assessment session.
func (r *PgRepository) Create(ctx context.Context, params CreateSessionParams) (*AssessmentSession, error) {
	const query = `
		INSERT INTO assessment_sessions (
			tenant_id, school_id, class_id, learning_area_id,
			academic_term_id, academic_year_id, name, evaluation_method,
			max_points, grading_scale_profile_id, scheduled_date, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::assessment_evaluation_method,
		        $9, $10, $11, $12)
		RETURNING id
	`
	var id string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		params.TenantID, params.SchoolID, params.ClassID, params.LearningAreaID,
		params.AcademicTermID, params.AcademicYearID, params.Name, params.EvaluationMethod,
		params.MaxPoints, params.GradingScaleProfileID, params.ScheduledDate, params.CreatedBy,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("assessments.Repository.Create: %s: %w", pgErr.Code, err)
		}
		return nil, fmt.Errorf("assessments.Repository.Create: %w", err)
	}
	return r.GetByID(ctx, id, params.TenantID, params.SchoolID)
}

// Update modifies an existing DRAFT session.
// max_points changes are blocked by DB trigger if any student_assessment_scores exist.
func (r *PgRepository) Update(ctx context.Context, params UpdateSessionParams) (*AssessmentSession, error) {
	const query = `
		UPDATE assessment_sessions
		SET name = $1,
		    evaluation_method = $2::assessment_evaluation_method,
		    max_points = $3,
		    grading_scale_profile_id = $4,
		    scheduled_date = $5,
		    updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7 AND school_id = $8
	`
	_, err := database.FromContext(ctx, r.pool).Exec(ctx, query,
		params.Name, params.EvaluationMethod, params.MaxPoints,
		params.GradingScaleProfileID, params.ScheduledDate,
		params.ID, params.TenantID, params.SchoolID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "P0002" {
			return nil, fmt.Errorf("assessments.Repository.Update: %w", ErrMaxPointsLocked)
		}
		return nil, fmt.Errorf("assessments.Repository.Update: %w", err)
	}
	return r.GetByID(ctx, params.ID, params.TenantID, params.SchoolID)
}

// Delete removes a session. Caller is responsible for status guard.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		DELETE FROM assessment_sessions
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// Submit transitions DRAFT → PENDING_APPROVAL.
func (r *PgRepository) Submit(ctx context.Context, id, tenantID, schoolID, userID string) error {
	const query = `
		UPDATE assessment_sessions
		SET status = 'PENDING_APPROVAL',
		    submitted_by = $1,
		    rejection_comment = NULL,
		    updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND school_id = $4
		  AND status = 'DRAFT'
	`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, userID, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.Submit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.Submit: %w", ErrInvalidStatus)
	}
	return nil
}

// Approve transitions PENDING_APPROVAL → PUBLISHED. DB trigger refreshes summaries.
func (r *PgRepository) Approve(ctx context.Context, id, tenantID, schoolID, userID string) error {
	const query = `
		UPDATE assessment_sessions
		SET status = 'PUBLISHED',
		    approved_by = $1,
		    updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND school_id = $4
		  AND status = 'PENDING_APPROVAL'
	`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, userID, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.Approve: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.Approve: %w", ErrInvalidStatus)
	}
	return nil
}

// Reject transitions PENDING_APPROVAL → DRAFT with a required comment.
func (r *PgRepository) Reject(ctx context.Context, id, tenantID, schoolID, userID, comment string) error {
	const query = `
		UPDATE assessment_sessions
		SET status = 'DRAFT',
		    rejection_comment = $1,
		    submitted_by = NULL,
		    updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND school_id = $4
		  AND status = 'PENDING_APPROVAL'
	`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, comment, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.Reject: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.Reject: %w", ErrInvalidStatus)
	}
	return nil
}

// ─── Score Methods on PgRepository ──────────────────────────────────────

func (r *PgRepository) UpsertScores(ctx context.Context, sessionID, tenantID string, entries []ScoreEntryPayload) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return 0, fmt.Errorf("assessments.Repository.UpsertScores: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				r.logger.Warnw("assessments.Repository.UpsertScores: rollback error", "error", rbErr.Error())
			}
		}
	}()

	// Check for non-ACTIVE enrollments
	studentIDs := make([]string, 0, len(entries))
	for _, e := range entries {
		studentIDs = append(studentIDs, e.StudentID)
	}
	const nonActiveCheck = `SELECT s.id, COALESCE(e.status::text,'NOT_ENROLLED') AS status FROM unnest($1::uuid[]) AS s(id) LEFT JOIN cbc_student_enrollments e ON e.student_id = s.id AND e.tenant_id = $2 AND e.class_id = (SELECT class_id FROM assessment_sessions WHERE id = $3 AND tenant_id = $2) WHERE COALESCE(e.status::text,'NOT_ENROLLED') != 'ACTIVE'`
	rows, err := tx.Query(ctx, nonActiveCheck, studentIDs, tenantID, sessionID)
	if err != nil {
		return 0, fmt.Errorf("assessments.Repository.UpsertScores: check enrollment: %w", err)
	}
	defer rows.Close()
	var blocked []string
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			return 0, fmt.Errorf("assessments.Repository.UpsertScores: scan: %w", err)
		}
		blocked = append(blocked, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("assessments.Repository.UpsertScores: rows: %w", err)
	}
	rows.Close()
	if len(blocked) > 0 {
		return 0, fmt.Errorf("assessments.Repository.UpsertScores: %w: non-ACTIVE students %v", ErrNotEnrolledActive, blocked)
	}

	const upsert = `INSERT INTO student_assessment_scores (tenant_id, session_id, student_id, raw_score, enrollment_status) VALUES ($1, $2, $3, $4, (SELECT status FROM cbc_student_enrollments WHERE student_id = $3 AND tenant_id = $1 AND class_id = (SELECT class_id FROM assessment_sessions WHERE id = $2 AND tenant_id = $1) LIMIT 1)) ON CONFLICT (session_id, student_id) DO UPDATE SET raw_score = EXCLUDED.raw_score, updated_at = NOW()`
	count := 0
	for _, e := range entries {
		_, err := tx.Exec(ctx, upsert, tenantID, sessionID, e.StudentID, e.RawScore)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23514" {
				return 0, fmt.Errorf("assessments.Repository.UpsertScores: %w: student %s score out of range", ErrScoreOutOfRange, e.StudentID)
			}
			return 0, fmt.Errorf("assessments.Repository.UpsertScores: upsert %s: %w", e.StudentID, err)
		}
		count++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("assessments.Repository.UpsertScores: commit: %w", err)
	}
	committed = true
	return count, nil
}

func (r *PgRepository) ListScores(ctx context.Context, sessionID, tenantID string, page, limit int) (*ScoreListResult, error) {
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	const countQuery = `SELECT COUNT(*) FROM student_assessment_scores WHERE session_id = $1 AND tenant_id = $2`
	var total int
	if err := database.FromContext(ctx, r.pool).QueryRow(ctx, countQuery, sessionID, tenantID).Scan(&total); err != nil {
		return nil, fmt.Errorf("assessments.Repository.ListScores: count: %w", err)
	}
	const dataQuery = `SELECT id, tenant_id, session_id, student_id, raw_score, calculated_percentage, final_performance_level::text, enrollment_status, created_at, updated_at FROM student_assessment_scores WHERE session_id = $1 AND tenant_id = $2 ORDER BY created_at ASC LIMIT $3 OFFSET $4`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, dataQuery, sessionID, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.ListScores: query: %w", err)
	}
	defer rows.Close()
	var items []StudentScore
	for rows.Next() {
		var s StudentScore
		var perfLevel *string
		if err := rows.Scan(&s.ID, &s.TenantID, &s.SessionID, &s.StudentID, &s.RawScore, &s.CalculatedPercentage, &perfLevel, &s.EnrollmentStatus, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("assessments.Repository.ListScores: scan: %w", err)
		}
		s.FinalPerformanceLevel = perfLevel
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessments.Repository.ListScores: rows: %w", err)
	}
	if items == nil {
		items = []StudentScore{}
	}
	return &ScoreListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (r *PgRepository) ListGradingScaleProfiles(ctx context.Context, tenantID, schoolID string) ([]map[string]interface{}, error) {
	const q = `SELECT id, name FROM grading_scale_profiles WHERE tenant_id = $1 AND school_id = $2 ORDER BY name`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, q, tenantID, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"id": id, "name": name})
	}
	return out, rows.Err()
}

func (r *PgRepository) UpsertRubricOutcomes(ctx context.Context, sessionID, tenantID string, entries []RubricEntryPayload) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	tx, err := database.Begin(ctx, r.pool)
	if err != nil {
		return 0, fmt.Errorf("assessments.Repository.UpsertRubricOutcomes: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				r.logger.Warnw("UpsertRubricOutcomes: rollback", "error", rbErr.Error())
			}
		}
	}()
	const upsert = `INSERT INTO student_assessment_outcome_grades (tenant_id, session_id, student_id, performance_indicator_id, awarded_level) VALUES ($1, $2, $3, $4, $5::cbc_performance_level) ON CONFLICT (session_id, student_id, performance_indicator_id) DO UPDATE SET awarded_level = EXCLUDED.awarded_level, updated_at = NOW()`
	count := 0
	for _, e := range entries {
		_, err := tx.Exec(ctx, upsert, tenantID, sessionID, e.StudentID, e.PerformanceIndicatorID, e.AwardedLevel)
		if err != nil {
			return 0, fmt.Errorf("assessments.Repository.UpsertRubricOutcomes: upsert: %w", err)
		}
		count++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("assessments.Repository.UpsertRubricOutcomes: commit: %w", err)
	}
	committed = true
	return count, nil
}

func (r *PgRepository) ListRubricOutcomes(ctx context.Context, sessionID, tenantID string) ([]RubricOutcome, error) {
	const q = `SELECT id, tenant_id, session_id, student_id, performance_indicator_id, awarded_level::text, created_at, updated_at FROM student_assessment_outcome_grades WHERE session_id = $1 AND tenant_id = $2`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, q, sessionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.ListRubricOutcomes: %w", err)
	}
	defer rows.Close()
	var out []RubricOutcome
	for rows.Next() {
		var o RubricOutcome
		if err := rows.Scan(&o.ID, &o.TenantID, &o.SessionID, &o.StudentID, &o.PerformanceIndicatorID, &o.AwardedLevel, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("assessments.Repository.ListRubricOutcomes: scan: %w", err)
		}
		out = append(out, o)
	}
	if out == nil {
		out = []RubricOutcome{}
	}
	return out, rows.Err()
}
