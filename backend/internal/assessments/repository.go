package assessments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles assessment database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ── Helpers ──────────────────────────────────────────────────────────────

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23P01"
	}
	return false
}

// IsTermFinalised checks whether an academic term has is_final = true.
func (r *PgRepository) IsTermFinalised(ctx context.Context, termID string) (bool, error) {
	const query = `SELECT is_final FROM academic_terms WHERE id = $1`
	var isFinal bool
	err := r.pool.QueryRow(ctx, query, termID).Scan(&isFinal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, fmt.Errorf("assessments.Repository.IsTermFinalised: %w", ErrNotFound)
		}
		return false, fmt.Errorf("assessments.Repository.IsTermFinalised: %w", err)
	}
	return isFinal, nil
}

// getTenantSchoolFromProfile verifies a profile belongs to a tenant/school and returns its ID.
func (r *PgRepository) getTenantSchoolFromProfile(ctx context.Context, profileID, tenantID, schoolID string) error {
	const query = `
		SELECT 1 FROM grading_scale_profiles
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var exists int
	err := r.pool.QueryRow(ctx, query, profileID, tenantID, schoolID).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("assessments.Repository.getTenantSchoolFromProfile: %w", ErrNotFound)
		}
		return fmt.Errorf("assessments.Repository.getTenantSchoolFromProfile: %w", err)
	}
	return nil
}

// ============================================================================
// GRADING SCALE PROFILES
// ============================================================================

// CreateScaleProfile inserts a new grading scale profile.
func (r *PgRepository) CreateScaleProfile(ctx context.Context, params CreateScaleProfileParams) (string, error) {
	const query = `
		INSERT INTO grading_scale_profiles (tenant_id, school_id, name)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, params.TenantID, params.SchoolID, params.Name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("assessments.Repository.CreateScaleProfile: %w", err)
	}
	return id, nil
}

// GetScaleProfileByID retrieves a single scale profile by ID.
func (r *PgRepository) GetScaleProfileByID(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfile, error) {
	const query = `
		SELECT id, tenant_id, school_id, name, is_active, created_at, updated_at
		FROM grading_scale_profiles
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var p ScaleProfile
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).
		Scan(&p.ID, &p.TenantID, &p.SchoolID, &p.Name, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assessments.Repository.GetScaleProfileByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("assessments.Repository.GetScaleProfileByID: %w", err)
	}
	return &p, nil
}

// ListScaleProfiles returns all scale profiles for a tenant/school.
func (r *PgRepository) ListScaleProfiles(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]ScaleProfile, error) {
	baseQuery := `
		SELECT id, tenant_id, school_id, name, is_active, created_at, updated_at
		FROM grading_scale_profiles
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []interface{}{tenantID, schoolID}
	if activeOnly {
		baseQuery += ` AND is_active = true`
	}
	baseQuery += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.ListScaleProfiles: %w", err)
	}
	defer rows.Close()

	var profiles []ScaleProfile
	for rows.Next() {
		var p ScaleProfile
		if err := rows.Scan(&p.ID, &p.TenantID, &p.SchoolID, &p.Name, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("assessments.Repository.ListScaleProfiles: %w", err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// ToggleScaleProfileActive toggles the is_active flag on a profile.
func (r *PgRepository) ToggleScaleProfileActive(ctx context.Context, id, tenantID, schoolID string, isActive bool) error {
	const query = `
		UPDATE grading_scale_profiles
		SET is_active = $4
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID, isActive)
	if err != nil {
		return fmt.Errorf("assessments.Repository.ToggleScaleProfileActive: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.ToggleScaleProfileActive: %w", ErrNotFound)
	}
	return nil
}

// DeleteScaleProfile removes a scale profile (only if no sessions reference it).
func (r *PgRepository) DeleteScaleProfile(ctx context.Context, id, tenantID, schoolID string) error {
	count, err := r.CountSessionsReferencingScale(ctx, id)
	if err != nil {
		return fmt.Errorf("assessments.Repository.DeleteScaleProfile: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("assessments.Repository.DeleteScaleProfile: %w", ErrScaleReferenced)
	}

	const query = `
		DELETE FROM grading_scale_profiles
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.DeleteScaleProfile: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.DeleteScaleProfile: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// GRADING SCALE RANGES
// ============================================================================

// CreateScaleRange inserts a new range into a profile.
func (r *PgRepository) CreateScaleRange(ctx context.Context, params CreateScaleRangeParams) (string, error) {
	const query = `
		INSERT INTO grading_scale_ranges (profile_id, performance_level, min_percentage, max_percentage, default_percentage_mapping)
		VALUES ($1, $2::cbc_performance_level, $3, $4, $5)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		params.ProfileID,
		params.PerformanceLevel,
		params.MinPercentage,
		params.MaxPercentage,
		params.DefaultPercentageMapping,
	).Scan(&id)
	if err != nil {
		if isExclusionViolation(err) {
			return "", fmt.Errorf("assessments.Repository.CreateScaleRange: range overlaps with existing range: %w", ErrConflict)
		}
		if isUniqueViolation(err) {
			return "", fmt.Errorf("assessments.Repository.CreateScaleRange: duplicate performance level for this profile: %w", ErrAlreadyExists)
		}
		return "", fmt.Errorf("assessments.Repository.CreateScaleRange: %w", err)
	}
	return id, nil
}

// GetScaleRangesByProfile returns all ranges for a profile.
func (r *PgRepository) GetScaleRangesByProfile(ctx context.Context, profileID, tenantID, schoolID string) ([]ScaleRange, error) {
	// First verify the profile belongs to the tenant/school
	if err := r.getTenantSchoolFromProfile(ctx, profileID, tenantID, schoolID); err != nil {
		return nil, err
	}

	const query = `
		SELECT id, profile_id, performance_level::text, min_percentage, max_percentage, default_percentage_mapping
		FROM grading_scale_ranges
		WHERE profile_id = $1
		ORDER BY min_percentage ASC
	`
	rows, err := r.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.GetScaleRangesByProfile: %w", err)
	}
	defer rows.Close()

	var ranges []ScaleRange
	for rows.Next() {
		var sr ScaleRange
		if err := rows.Scan(&sr.ID, &sr.ProfileID, &sr.PerformanceLevel, &sr.MinPercentage, &sr.MaxPercentage, &sr.DefaultPercentageMapping); err != nil {
			return nil, fmt.Errorf("assessments.Repository.GetScaleRangesByProfile: %w", err)
		}
		ranges = append(ranges, sr)
	}
	return ranges, nil
}

// DeleteScaleRange removes a single range from a profile.
func (r *PgRepository) DeleteScaleRange(ctx context.Context, rangeID, profileID, tenantID, schoolID string) error {
	if err := r.getTenantSchoolFromProfile(ctx, profileID, tenantID, schoolID); err != nil {
		return err
	}

	const query = `
		DELETE FROM grading_scale_ranges
		WHERE id = $1 AND profile_id = $2
	`
	result, err := r.pool.Exec(ctx, query, rangeID, profileID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.DeleteScaleRange: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.DeleteScaleRange: %w", ErrNotFound)
	}
	return nil
}

// BulkSetScaleRanges replaces all ranges for a profile in a transaction.
func (r *PgRepository) BulkSetScaleRanges(ctx context.Context, profileID string, ranges []CreateScaleRangeParams) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.BulkSetScaleRanges: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Delete existing ranges
	_, err = tx.Exec(ctx, `DELETE FROM grading_scale_ranges WHERE profile_id = $1`, profileID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.BulkSetScaleRanges: delete existing: %w", err)
	}

	// Insert new ranges
	ids := make([]string, 0, len(ranges))
	for _, sr := range ranges {
		const query = `
			INSERT INTO grading_scale_ranges (profile_id, performance_level, min_percentage, max_percentage, default_percentage_mapping)
			VALUES ($1, $2::cbc_performance_level, $3, $4, $5)
			RETURNING id
		`
		var id string
		err := tx.QueryRow(ctx, query,
			profileID,
			sr.PerformanceLevel,
			sr.MinPercentage,
			sr.MaxPercentage,
			sr.DefaultPercentageMapping,
		).Scan(&id)
		if err != nil {
			if isExclusionViolation(err) {
				return nil, fmt.Errorf("assessments.Repository.BulkSetScaleRanges: overlapping ranges: %w", ErrConflict)
			}
			return nil, fmt.Errorf("assessments.Repository.BulkSetScaleRanges: %w", err)
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("assessments.Repository.BulkSetScaleRanges: commit: %w", err)
	}
	return ids, nil
}

// ============================================================================
// ASSESSMENT SESSIONS
// ============================================================================

// GetSessionStatusAndTerm returns just the status and term_id for a session (lightweight, no school filter).
func (r *PgRepository) GetSessionStatusAndTerm(ctx context.Context, id, tenantID string) (string, string, error) {
	const query = `
		SELECT status::text, academic_term_id
		FROM assessment_sessions
		WHERE id = $1 AND tenant_id = $2
	`
	var status, termID string
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(&status, &termID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", fmt.Errorf("assessments.Repository.GetSessionStatusAndTerm: %w", ErrNotFound)
		}
		return "", "", fmt.Errorf("assessments.Repository.GetSessionStatusAndTerm: %w", err)
	}
	return status, termID, nil
}

// CreateSession inserts a new assessment session.
func (r *PgRepository) CreateSession(ctx context.Context, params CreateSessionParams) (string, error) {
	var scheduledDate interface{}
	if params.ScheduledDate != nil && *params.ScheduledDate != "" {
		parsed, err := time.Parse("2006-01-02", *params.ScheduledDate)
		if err == nil {
			scheduledDate = parsed
		}
	}

	const query = `
		INSERT INTO assessment_sessions (
			tenant_id, school_id, class_id, learning_area_id,
			academic_term_id, academic_year_id, name, evaluation_method,
			max_points, grading_scale_profile_id, created_by, scheduled_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::assessment_evaluation_method,
		          $9, $10, $11, $12)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.SchoolID,
		params.ClassID,
		params.LearningAreaID,
		params.AcademicTermID,
		params.AcademicYearID,
		params.Name,
		params.EvaluationMethod,
		params.MaxPoints,
		params.GradingScaleProfileID,
		params.CreatedBy,
		scheduledDate,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("assessments.Repository.CreateSession: %w", err)
	}
	return id, nil
}

// GetSessionByID retrieves a single session by ID.
func (r *PgRepository) GetSessionByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error) {
	const query = `
		SELECT id, tenant_id, school_id, class_id, learning_area_id,
		       academic_term_id, academic_year_id, name,
		       evaluation_method::text, max_points, grading_scale_profile_id,
		       status::text, rejection_comment, submitted_by, approved_by,
		       scheduled_date, created_at, updated_at, created_by
		FROM assessment_sessions
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var s AssessmentSession
	var scheduledDate *time.Time
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).
		Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.LearningAreaID,
			&s.AcademicTermID, &s.AcademicYearID, &s.Name,
			&s.EvaluationMethod, &s.MaxPoints, &s.GradingScaleProfileID,
			&s.Status, &s.RejectionComment, &s.SubmittedBy, &s.ApprovedBy,
			&scheduledDate, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
		)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assessments.Repository.GetSessionByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("assessments.Repository.GetSessionByID: %w", err)
	}
	if scheduledDate != nil {
		dateStr := scheduledDate.Format("2006-01-02")
		s.ScheduledDate = &dateStr
	}
	return &s, nil
}

// ListSessions returns paginated assessment sessions with optional filters.
func (r *PgRepository) ListSessions(ctx context.Context, tenantID, schoolID string, filters SessionFilters) ([]AssessmentSession, int, error) {
	where := []string{"s.tenant_id = $1", "s.school_id = $2"}
	args := []interface{}{tenantID, schoolID}
	argIdx := 3

	if filters.ClassID != nil && *filters.ClassID != "" {
		where = append(where, fmt.Sprintf("s.class_id = $%d", argIdx))
		args = append(args, *filters.ClassID)
		argIdx++
	}
	if filters.LearningAreaID != nil && *filters.LearningAreaID != "" {
		where = append(where, fmt.Sprintf("s.learning_area_id = $%d", argIdx))
		args = append(args, *filters.LearningAreaID)
		argIdx++
	}
	if filters.AcademicTermID != nil && *filters.AcademicTermID != "" {
		where = append(where, fmt.Sprintf("s.academic_term_id = $%d", argIdx))
		args = append(args, *filters.AcademicTermID)
		argIdx++
	}
	if filters.Status != nil && *filters.Status != "" {
		where = append(where, fmt.Sprintf("s.status = $%d::assessment_session_status", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.EvaluationMethod != nil && *filters.EvaluationMethod != "" {
		where = append(where, fmt.Sprintf("s.evaluation_method = $%d::assessment_evaluation_method", argIdx))
		args = append(args, *filters.EvaluationMethod)
		argIdx++
	}
	if filters.Search != "" {
		where = append(where, fmt.Sprintf("s.name ILIKE $%d", argIdx))
		args = append(args, "%"+filters.Search+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	countQuery := `SELECT COUNT(*) FROM assessment_sessions s WHERE ` + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("assessments.Repository.ListSessions: count: %w", err)
	}

	// Paginate
	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	selectQuery := `
		SELECT s.id, s.tenant_id, s.school_id, s.class_id, s.learning_area_id,
		       s.academic_term_id, s.academic_year_id, s.name,
		       s.evaluation_method::text, s.max_points, s.grading_scale_profile_id,
		       s.status::text, s.rejection_comment, s.submitted_by, s.approved_by,
		       s.scheduled_date, s.created_at, s.updated_at, s.created_by
		FROM assessment_sessions s
		WHERE ` + whereClause + `
		ORDER BY s.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("assessments.Repository.ListSessions: %w", err)
	}
	defer rows.Close()

	var sessions []AssessmentSession
	for rows.Next() {
		var s AssessmentSession
		var scheduledDate *time.Time
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SchoolID, &s.ClassID, &s.LearningAreaID,
			&s.AcademicTermID, &s.AcademicYearID, &s.Name,
			&s.EvaluationMethod, &s.MaxPoints, &s.GradingScaleProfileID,
			&s.Status, &s.RejectionComment, &s.SubmittedBy, &s.ApprovedBy,
			&scheduledDate, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("assessments.Repository.ListSessions: scan: %w", err)
		}
		if scheduledDate != nil {
			dateStr := scheduledDate.Format("2006-01-02")
			s.ScheduledDate = &dateStr
		}
		sessions = append(sessions, s)
	}
	return sessions, total, nil
}

// UpdateSessionStatus updates the status and related fields of a session.
func (r *PgRepository) UpdateSessionStatus(ctx context.Context, id, tenantID, schoolID string, status string, rejectionComment *string, approvedBy *string) error {
	var query string
	var args []interface{}

	switch status {
	case "PENDING_APPROVAL":
		query = `
			UPDATE assessment_sessions
			SET status = $4::assessment_session_status, rejection_comment = NULL, submitted_by = $5,
			    approved_by = NULL
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		`
		args = []interface{}{id, tenantID, schoolID, status, approvedBy}
	case "PUBLISHED":
		query = `
			UPDATE assessment_sessions
			SET status = $4::assessment_session_status, approved_by = $5
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		`
		args = []interface{}{id, tenantID, schoolID, status, approvedBy}
	case "DRAFT":
		query = `
			UPDATE assessment_sessions
			SET status = $4::assessment_session_status, rejection_comment = $5,
			    submitted_by = NULL, approved_by = NULL
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		`
		args = []interface{}{id, tenantID, schoolID, status, rejectionComment}
	default:
		query = `
			UPDATE assessment_sessions
			SET status = $4::assessment_session_status
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		`
		args = []interface{}{id, tenantID, schoolID, status}
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("assessments.Repository.UpdateSessionStatus: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("assessments.Repository.UpdateSessionStatus: %w", ErrNotFound)
	}
	return nil
}

// HasScoresForSession checks whether any student scores exist for a session.
func (r *PgRepository) HasScoresForSession(ctx context.Context, sessionID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM student_assessment_scores WHERE session_id = $1
			UNION
			SELECT 1 FROM student_assessment_outcome_grades WHERE session_id = $1
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, sessionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("assessments.Repository.HasScoresForSession: %w", err)
	}
	return exists, nil
}

// CountSessionsReferencingScale counts sessions that reference a scale profile.
func (r *PgRepository) CountSessionsReferencingScale(ctx context.Context, profileID string) (int, error) {
	const query = `
		SELECT COUNT(*) FROM assessment_sessions
		WHERE grading_scale_profile_id = $1
	`
	var count int
	err := r.pool.QueryRow(ctx, query, profileID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("assessments.Repository.CountSessionsReferencingScale: %w", err)
	}
	return count, nil
}

// ============================================================================
// STUDENT SCORES (Quantitative)
// ============================================================================

// UpsertStudentScore inserts or updates a student's score for a session.
func (r *PgRepository) UpsertStudentScore(ctx context.Context, params UpsertScoreParams) error {
	var calculatedPercentage *float64
	if params.RawScore != nil {
		// Calculate percentage from max_points
		const maxPointsQuery = `SELECT max_points FROM assessment_sessions WHERE id = $1`
		var maxPoints *float64
		err := r.pool.QueryRow(ctx, maxPointsQuery, params.SessionID).Scan(&maxPoints)
		if err != nil {
			return fmt.Errorf("assessments.Repository.UpsertStudentScore: get max_points: %w", err)
		}
		if maxPoints != nil && *maxPoints > 0 {
			pct := (*params.RawScore / *maxPoints) * 100
			calculatedPercentage = &pct
		}
	}

	const query = `
		INSERT INTO student_assessment_scores (tenant_id, session_id, student_id, raw_score, calculated_percentage, enrollment_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (session_id, student_id)
		DO UPDATE SET raw_score = EXCLUDED.raw_score,
		              calculated_percentage = EXCLUDED.calculated_percentage
		WHERE student_assessment_scores.final_performance_level IS NULL
	`
	_, err := r.pool.Exec(ctx, query,
		params.TenantID,
		params.SessionID,
		params.StudentID,
		params.RawScore,
		calculatedPercentage,
		params.EnrollmentStatus,
	)
	if err != nil {
		return fmt.Errorf("assessments.Repository.UpsertStudentScore: %w", err)
	}
	return nil
}

// BulkUpsertStudentScores bulk-upserts student scores for a session.
func (r *PgRepository) BulkUpsertStudentScores(ctx context.Context, params []UpsertScoreParams) error {
	if len(params) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("assessments.Repository.BulkUpsertStudentScores: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Get max_points for the session
	var maxPoints *float64
	if len(params) > 0 {
		err := tx.QueryRow(ctx, `SELECT max_points FROM assessment_sessions WHERE id = $1`, params[0].SessionID).Scan(&maxPoints)
		if err != nil {
			return fmt.Errorf("assessments.Repository.BulkUpsertStudentScores: get max_points: %w", err)
		}
	}

	for _, p := range params {
		var cp *float64
		if p.RawScore != nil && maxPoints != nil && *maxPoints > 0 {
			pct := (*p.RawScore / *maxPoints) * 100
			cp = &pct
		}

		const query = `
			INSERT INTO student_assessment_scores (tenant_id, session_id, student_id, raw_score, calculated_percentage, enrollment_status)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (session_id, student_id)
			DO UPDATE SET raw_score = EXCLUDED.raw_score,
			              calculated_percentage = EXCLUDED.calculated_percentage
			WHERE student_assessment_scores.final_performance_level IS NULL
		`
		_, err := tx.Exec(ctx, query,
			p.TenantID,
			p.SessionID,
			p.StudentID,
			p.RawScore,
			cp,
			p.EnrollmentStatus,
		)
		if err != nil {
			return fmt.Errorf("assessments.Repository.BulkUpsertStudentScores: upsert: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("assessments.Repository.BulkUpsertStudentScores: commit: %w", err)
	}
	return nil
}

// GetStudentScoresBySession returns all scores for a session.
func (r *PgRepository) GetStudentScoresBySession(ctx context.Context, sessionID, tenantID, schoolID string) ([]StudentScore, error) {
	const query = `
		SELECT sas.id, sas.session_id, sas.student_id,
		       sas.raw_score, sas.calculated_percentage,
		       sas.final_performance_level::text, sas.enrollment_status
		FROM student_assessment_scores sas
		JOIN assessment_sessions s ON s.id = sas.session_id
		WHERE sas.session_id = $1 AND s.tenant_id = $2 AND s.school_id = $3
		ORDER BY sas.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, sessionID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.GetStudentScoresBySession: %w", err)
	}
	defer rows.Close()

	var scores []StudentScore
	for rows.Next() {
		var sc StudentScore
		var fpl *string
		if err := rows.Scan(&sc.ID, &sc.SessionID, &sc.StudentID,
			&sc.RawScore, &sc.CalculatedPercentage,
			&fpl, &sc.EnrollmentStatus,
		); err != nil {
			return nil, fmt.Errorf("assessments.Repository.GetStudentScoresBySession: scan: %w", err)
		}
		if fpl != nil && *fpl != "" {
			sc.FinalPerformanceLevel = fpl
		}
		scores = append(scores, sc)
	}
	return scores, nil
}

// SnapshotPerformanceLevels computes and writes final_performance_level for all scores in a session.
func (r *PgRepository) SnapshotPerformanceLevels(ctx context.Context, sessionID string, profile *ScaleProfile) error {
	// Get the ranges
	const rangeQuery = `
		SELECT performance_level::text, min_percentage, max_percentage, default_percentage_mapping
		FROM grading_scale_ranges
		WHERE profile_id = $1
		ORDER BY min_percentage ASC
	`
	rows, err := r.pool.Query(ctx, rangeQuery, profile.ID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.SnapshotPerformanceLevels: get ranges: %w", err)
	}
	defer rows.Close()

	// Update each score's final_performance_level
	const updateQuery = `
		UPDATE student_assessment_scores
		SET final_performance_level = (
			SELECT level::cbc_performance_level FROM (
				SELECT r.performance_level::text AS level
				FROM grading_scale_ranges r
				WHERE r.profile_id = $2
				  AND student_assessment_scores.calculated_percentage >= r.min_percentage
				  AND student_assessment_scores.calculated_percentage <= r.max_percentage
				LIMIT 1
			) sub
		)
		WHERE session_id = $1
		  AND final_performance_level IS NULL
		  AND calculated_percentage IS NOT NULL
	`
	_, err = r.pool.Exec(ctx, updateQuery, sessionID, profile.ID)
	if err != nil {
		return fmt.Errorf("assessments.Repository.SnapshotPerformanceLevels: update: %w", err)
	}
	return nil
}

// ============================================================================
// STUDENT OUTCOME GRADES (Rubric)
// ============================================================================

// UpsertOutcomeGrade inserts or updates a rubric outcome grade.
func (r *PgRepository) UpsertOutcomeGrade(ctx context.Context, params UpsertOutcomeGradeParams) error {
	const query = `
		INSERT INTO student_assessment_outcome_grades (tenant_id, session_id, student_id, performance_indicator_id, awarded_level)
		VALUES ($1, $2, $3, $4, $5::cbc_performance_level)
		ON CONFLICT (session_id, student_id, performance_indicator_id)
		DO UPDATE SET awarded_level = EXCLUDED.awarded_level
	`
	_, err := r.pool.Exec(ctx, query,
		params.TenantID,
		params.SessionID,
		params.StudentID,
		params.PerformanceIndicatorID,
		params.AwardedLevel,
	)
	if err != nil {
		return fmt.Errorf("assessments.Repository.UpsertOutcomeGrade: %w", err)
	}
	return nil
}

// BulkUpsertOutcomeGrades bulk-upserts rubric outcome grades.
func (r *PgRepository) BulkUpsertOutcomeGrades(ctx context.Context, params []UpsertOutcomeGradeParams) error {
	if len(params) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("assessments.Repository.BulkUpsertOutcomeGrades: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, p := range params {
		const query = `
			INSERT INTO student_assessment_outcome_grades (tenant_id, session_id, student_id, performance_indicator_id, awarded_level)
			VALUES ($1, $2, $3, $4, $5::cbc_performance_level)
			ON CONFLICT (session_id, student_id, performance_indicator_id)
			DO UPDATE SET awarded_level = EXCLUDED.awarded_level
		`
		_, err := tx.Exec(ctx, query,
			p.TenantID,
			p.SessionID,
			p.StudentID,
			p.PerformanceIndicatorID,
			p.AwardedLevel,
		)
		if err != nil {
			return fmt.Errorf("assessments.Repository.BulkUpsertOutcomeGrades: upsert: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("assessments.Repository.BulkUpsertOutcomeGrades: commit: %w", err)
	}
	return nil
}

// GetOutcomeGradesBySession returns all outcome grades for a session.
func (r *PgRepository) GetOutcomeGradesBySession(ctx context.Context, sessionID, tenantID, schoolID string) ([]OutcomeGrade, error) {
	const query = `
		SELECT sog.id, sog.session_id, sog.student_id,
		       sog.performance_indicator_id, sog.awarded_level::text
		FROM student_assessment_outcome_grades sog
		JOIN assessment_sessions s ON s.id = sog.session_id
		WHERE sog.session_id = $1 AND s.tenant_id = $2 AND s.school_id = $3
		ORDER BY sog.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, sessionID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.GetOutcomeGradesBySession: %w", err)
	}
	defer rows.Close()

	var grades []OutcomeGrade
	for rows.Next() {
		var g OutcomeGrade
		if err := rows.Scan(&g.ID, &g.SessionID, &g.StudentID, &g.PerformanceIndicatorID, &g.AwardedLevel); err != nil {
			return nil, fmt.Errorf("assessments.Repository.GetOutcomeGradesBySession: scan: %w", err)
		}
		grades = append(grades, g)
	}
	return grades, nil
}

// GetOutcomeGradesByStudent returns outcome grades for a specific student in a session.
func (r *PgRepository) GetOutcomeGradesByStudent(ctx context.Context, sessionID, studentID string) ([]OutcomeGrade, error) {
	const query = `
		SELECT id, session_id, student_id, performance_indicator_id, awarded_level::text
		FROM student_assessment_outcome_grades
		WHERE session_id = $1 AND student_id = $2
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, sessionID, studentID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.GetOutcomeGradesByStudent: %w", err)
	}
	defer rows.Close()

	var grades []OutcomeGrade
	for rows.Next() {
		var g OutcomeGrade
		if err := rows.Scan(&g.ID, &g.SessionID, &g.StudentID, &g.PerformanceIndicatorID, &g.AwardedLevel); err != nil {
			return nil, fmt.Errorf("assessments.Repository.GetOutcomeGradesByStudent: scan: %w", err)
		}
		grades = append(grades, g)
	}
	return grades, nil
}

// ============================================================================
// REPORT CARD AGGREGATION & PARENT VIEW
// ============================================================================

// GetStudentTermGrades implements the "Last One" chronological mode aggregator.
// For each learning area, it collects all published performance levels,
// finds the mode (most frequent), and breaks ties by selecting the
// chronologically latest assessment among the tied levels.
func (r *PgRepository) GetStudentTermGrades(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
	const query = `
		WITH session_scores AS (
			-- Quantitative scores with converted levels
			SELECT
				s.learning_area_id,
				la.name AS learning_area_name,
				la.code AS learning_area_code,
				sas.final_performance_level::text AS level,
				s.created_at AS assessment_date
			FROM student_assessment_scores sas
			JOIN assessment_sessions s ON s.id = sas.session_id
			JOIN cbc_learning_areas la ON la.id = s.learning_area_id
			WHERE sas.student_id = $3
			  AND s.academic_term_id = $4
			  AND s.status = 'PUBLISHED'
			  AND sas.final_performance_level IS NOT NULL
			  AND s.tenant_id = $1 AND s.school_id = $2

			UNION ALL

			-- Rubric scores
			SELECT
				s.learning_area_id,
				la.name AS learning_area_name,
				la.code AS learning_area_code,
				sog.awarded_level::text AS level,
				s.created_at AS assessment_date
			FROM student_assessment_outcome_grades sog
			JOIN assessment_sessions s ON s.id = sog.session_id
			JOIN cbc_learning_areas la ON la.id = s.learning_area_id
			WHERE sog.student_id = $3
			  AND s.academic_term_id = $4
			  AND s.status = 'PUBLISHED'
			  AND s.tenant_id = $1 AND s.school_id = $2
		),
		level_counts AS (
			SELECT
				learning_area_id,
				learning_area_name,
				learning_area_code,
				level,
				COUNT(*) AS cnt,
				MAX(assessment_date) AS latest_date
			FROM session_scores
			GROUP BY learning_area_id, learning_area_name, learning_area_code, level
		),
		max_counts AS (
			SELECT
				learning_area_id,
				MAX(cnt) AS max_cnt
			FROM level_counts
			GROUP BY learning_area_id
		),
		tied_levels AS (
			SELECT
				lc.learning_area_id,
				lc.learning_area_name,
				lc.learning_area_code,
				lc.level,
				lc.cnt,
				lc.latest_date
			FROM level_counts lc
			JOIN max_counts mc ON mc.learning_area_id = lc.learning_area_id
			WHERE lc.cnt = mc.max_cnt
		),
		ranked AS (
			SELECT
				learning_area_id,
				learning_area_name,
				learning_area_code,
				level,
				cnt,
				latest_date,
				ROW_NUMBER() OVER (
					PARTITION BY learning_area_id
					ORDER BY latest_date DESC, level DESC
				) AS rn
			FROM tied_levels
		)
		SELECT
			learning_area_id,
			learning_area_name,
			learning_area_code,
			COALESCE(level, 'BE') AS final_level,
			cnt AS assessment_count
		FROM ranked
		WHERE rn = 1
		ORDER BY learning_area_name ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.GetStudentTermGrades: %w", err)
	}
	defer rows.Close()

	var grades []StudentTermGrade
	for rows.Next() {
		var g StudentTermGrade
		if err := rows.Scan(&g.LearningAreaID, &g.LearningAreaName, &g.LearningAreaCode, &g.FinalLevel, &g.AssessmentCount); err != nil {
			return nil, fmt.Errorf("assessments.Repository.GetStudentTermGrades: scan: %w", err)
		}
		grades = append(grades, g)
	}
	return grades, nil
}

// GetPublishedSessionsForParent returns all published sessions for a student in a term.
func (r *PgRepository) GetPublishedSessionsForParent(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]ParentAssessmentView, error) {
	const query = `
		SELECT
			s.id AS session_id,
			s.name AS session_name,
			s.evaluation_method::text,
			s.scheduled_date,
			sas.raw_score,
			s.max_points,
			sas.final_performance_level::text AS performance_level
		FROM assessment_sessions s
		LEFT JOIN student_assessment_scores sas
			ON sas.session_id = s.id AND sas.student_id = $3
		WHERE s.academic_term_id = $4
		  AND s.status = 'PUBLISHED'
		  AND s.tenant_id = $1 AND s.school_id = $2
		  AND (sas.id IS NOT NULL OR s.evaluation_method = 'RUBRIC')
		ORDER BY s.scheduled_date ASC NULLS LAST, s.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.GetPublishedSessionsForParent: %w", err)
	}
	defer rows.Close()

	var views []ParentAssessmentView
	for rows.Next() {
		var v ParentAssessmentView
		var scheduledDate *time.Time
		if err := rows.Scan(
			&v.SessionID, &v.SessionName, &v.EvaluationMethod,
			&scheduledDate, &v.RawScore, &v.MaxPoints, &v.PerformanceLevel,
		); err != nil {
			return nil, fmt.Errorf("assessments.Repository.GetPublishedSessionsForParent: scan: %w", err)
		}
		if scheduledDate != nil {
			ds := scheduledDate.Format("2006-01-02")
			v.ScheduledDate = &ds
		}
		views = append(views, v)
	}
	return views, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ASSESSMENT WEIGHT CONFIGS
// ═══════════════════════════════════════════════════════════════════════════

// ListWeightConfigs returns assessment weight configs matching the given filter.
func (r *PgRepository) ListWeightConfigs(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
	query := `SELECT id, grade_level::text, assessment_type_code, target_exam, weight_percent, effective_from, notes, created_at FROM assessment_weight_configs WHERE 1=1`
	var args []interface{}
	argIdx := 1

	if filter.GradeLevel != nil {
		query += fmt.Sprintf(` AND grade_level = $%d`, argIdx)
		args = append(args, *filter.GradeLevel)
		argIdx++
	}
	if filter.TargetExam != nil {
		query += fmt.Sprintf(` AND target_exam = $%d`, argIdx)
		args = append(args, *filter.TargetExam)
		argIdx++
	}
	if filter.EffectiveFrom != nil {
		query += fmt.Sprintf(` AND effective_from = $%d`, argIdx)
		args = append(args, *filter.EffectiveFrom)
	}

	query += ` ORDER BY grade_level, effective_from DESC, assessment_type_code`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("assessments.Repository.ListWeightConfigs: %w", err)
	}
	defer rows.Close()

	var items []AssessmentWeightConfig
	for rows.Next() {
		var c AssessmentWeightConfig
		if err := rows.Scan(&c.ID, &c.GradeLevel, &c.AssessmentTypeCode, &c.TargetExam, &c.WeightPercent, &c.EffectiveFrom, &c.Notes, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("assessments.Repository.ListWeightConfigs: scan: %w", err)
		}
		items = append(items, c)
	}
	return items, nil
}

// GetWeightConfigByID returns a single weight config by ID.
func (r *PgRepository) GetWeightConfigByID(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
	const query = `SELECT id, grade_level::text, assessment_type_code, target_exam, weight_percent, effective_from, notes, created_at FROM assessment_weight_configs WHERE id = $1`
	var c AssessmentWeightConfig
	err := r.pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.GradeLevel, &c.AssessmentTypeCode, &c.TargetExam, &c.WeightPercent, &c.EffectiveFrom, &c.Notes, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("assessments.Repository.GetWeightConfigByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("assessments.Repository.GetWeightConfigByID: %w", err)
	}
	return &c, nil
}
