package assessment

import (
	"context"
	"errors"
	"fmt"

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

// isUniqueViolation checks if an error is a PostgreSQL unique constraint violation (23505).
// Falls back to string matching for testing with plain error strings.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	// Fallback for tests that pass plain errors
	msg := err.Error()
	return contains(msg, "unique constraint") || contains(msg, "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// isFKViolation checks if an error is a PostgreSQL foreign key violation (23503).
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

// ============================================================================
// Cross-Domain Resolver Implementations
// ============================================================================

// IsStudentInClass checks whether a student is enrolled in a given class.
// Implements assessment.ClassStudentResolver.
func (r *PgRepository) IsStudentInClass(ctx context.Context, studentID, classID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_student_enrollments
			WHERE student_id = $1 AND class_id = $2 AND status = 'ACTIVE'
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, studentID, classID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("assessment.Repository.IsStudentInClass: %w", err)
	}
	return exists, nil
}

// ============================================================================
// BLUEPRINTS
// ============================================================================

// CreateBlueprint inserts a new blueprint and returns its ID.
func (r *PgRepository) CreateBlueprint(ctx context.Context, bp *AssessmentBlueprint) (string, error) {
	const query = `
		INSERT INTO assessment_blueprints
			(tenant_id, school_id, title, type, grade_level, academic_year, term)
		VALUES ($1, $2, $3, $4, $5::cbc_grade_level, $6, $7)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		bp.TenantID, bp.SchoolID, bp.Title,
		bp.Type, bp.GradeLevel, bp.AcademicYear, bp.Term,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("assessment.Repository.CreateBlueprint: %w", err)
	}
	return id, nil
}

// GetBlueprintByID retrieves a single blueprint by primary key.
func (r *PgRepository) GetBlueprintByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
	const query = `
		SELECT id, tenant_id, school_id, title,
		       type::text, grade_level::text,
		       academic_year, term, created_at::text
		FROM assessment_blueprints
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var bp AssessmentBlueprint
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&bp.ID, &bp.TenantID, &bp.SchoolID, &bp.Title,
		&bp.Type, &bp.GradeLevel, &bp.AcademicYear, &bp.Term, &bp.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assessment.Repository.GetBlueprintByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("assessment.Repository.GetBlueprintByID: %w", err)
	}
	return &bp, nil
}

// ListBlueprints returns blueprints filtered by the given query parameters.
func (r *PgRepository) ListBlueprints(ctx context.Context, tenantID string, query ListBlueprintsQuery) ([]AssessmentBlueprint, error) {
	baseQuery := `
		SELECT id, tenant_id, school_id, title,
		       type::text, grade_level::text,
		       academic_year, term, created_at::text
		FROM assessment_blueprints
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if query.SchoolID != "" {
		baseQuery += fmt.Sprintf(" AND school_id = $%d", argIdx)
		args = append(args, query.SchoolID)
		argIdx++
	}
	if query.GradeLevel != "" {
		baseQuery += fmt.Sprintf(" AND grade_level = $%d::cbc_grade_level", argIdx)
		args = append(args, query.GradeLevel)
		argIdx++
	}
	if query.Term > 0 {
		baseQuery += fmt.Sprintf(" AND term = $%d", argIdx)
		args = append(args, query.Term)
		argIdx++
	}
	if query.AcademicYear > 0 {
		baseQuery += fmt.Sprintf(" AND academic_year = $%d", argIdx)
		args = append(args, query.AcademicYear)
	}

	baseQuery += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListBlueprints: %w", err)
	}
	defer rows.Close()

	var blueprints []AssessmentBlueprint
	for rows.Next() {
		var bp AssessmentBlueprint
		if err := rows.Scan(
			&bp.ID, &bp.TenantID, &bp.SchoolID, &bp.Title,
			&bp.Type, &bp.GradeLevel, &bp.AcademicYear, &bp.Term, &bp.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("assessment.Repository.ListBlueprints: scan: %w", err)
		}
		blueprints = append(blueprints, bp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListBlueprints: rows: %w", err)
	}
	if blueprints == nil {
		blueprints = []AssessmentBlueprint{}
	}
	return blueprints, nil
}

// UpdateBlueprint applies changes to a blueprint.
func (r *PgRepository) UpdateBlueprint(ctx context.Context, bp *AssessmentBlueprint) error {
	const query = `
		UPDATE assessment_blueprints
		SET title = $1, type = $2::cbc_assessment_type
		WHERE id = $3 AND tenant_id = $4 AND school_id = $5
	`
	tag, err := r.pool.Exec(ctx, query, bp.Title, bp.Type, bp.ID, bp.TenantID, bp.SchoolID)
	if err != nil {
		return fmt.Errorf("assessment.Repository.UpdateBlueprint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessment.Repository.UpdateBlueprint: %w", ErrNotFound)
	}
	return nil
}

// DeleteBlueprint removes a blueprint by ID.
func (r *PgRepository) DeleteBlueprint(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		DELETE FROM assessment_blueprints
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		// FK violation from assessment_sessions (ON DELETE RESTRICT)
		if isFKViolation(err) {
			return fmt.Errorf("assessment.Repository.DeleteBlueprint: %w", ErrConflict)
		}
		return fmt.Errorf("assessment.Repository.DeleteBlueprint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessment.Repository.DeleteBlueprint: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// BLUEPRINT DETAIL (with indicators)
// ============================================================================

// GetBlueprintDetail retrieves a blueprint with its linked indicators.
func (r *PgRepository) GetBlueprintDetail(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error) {
	// First fetch the blueprint itself
	bp, err := r.GetBlueprintByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.GetBlueprintDetail: %w", err)
	}

	// Then fetch linked indicators
	const indicatorsQuery = `
		SELECT pi.id, pi.description
		FROM assessment_blueprint_indicators abi
		JOIN performance_indicators pi ON pi.id = abi.indicator_id
		WHERE abi.blueprint_id = $1
		ORDER BY pi.sequence_order ASC
	`
	rows, err := r.pool.Query(ctx, indicatorsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.GetBlueprintDetail: indicators: %w", err)
	}
	defer rows.Close()

	var indicators []LinkedIndicator
	for rows.Next() {
		var li LinkedIndicator
		if err := rows.Scan(&li.ID, &li.Description); err != nil {
			return nil, fmt.Errorf("assessment.Repository.GetBlueprintDetail: scan indicator: %w", err)
		}
		indicators = append(indicators, li)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessment.Repository.GetBlueprintDetail: rows: %w", err)
	}
	if indicators == nil {
		indicators = []LinkedIndicator{}
	}

	return &BlueprintDetail{
		AssessmentBlueprint: *bp,
		Indicators:          indicators,
	}, nil
}

// ============================================================================
// BLUEPRINT ↔ INDICATOR LINKING
// ============================================================================

// LinkIndicators links multiple performance indicators to a blueprint.
func (r *PgRepository) LinkIndicators(ctx context.Context, blueprintID string, indicatorIDs []string) error {
	// Build bulk insert
	const query = `
		INSERT INTO assessment_blueprint_indicators (blueprint_id, indicator_id)
		VALUES ($1, $2)
		ON CONFLICT (blueprint_id, indicator_id) DO NOTHING
	`
	for _, indicatorID := range indicatorIDs {
		_, err := r.pool.Exec(ctx, query, blueprintID, indicatorID)
		if err != nil {
			// FK violation means either blueprint or indicator doesn't exist
			if isFKViolation(err) {
				return fmt.Errorf("assessment.Repository.LinkIndicators: %w", ErrNotFound)
			}
			return fmt.Errorf("assessment.Repository.LinkIndicators: %w", err)
		}
	}
	return nil
}

// UnlinkIndicator removes a single indicator from a blueprint.
func (r *PgRepository) UnlinkIndicator(ctx context.Context, blueprintID, indicatorID string) error {
	const query = `
		DELETE FROM assessment_blueprint_indicators
		WHERE blueprint_id = $1 AND indicator_id = $2
	`
	tag, err := r.pool.Exec(ctx, query, blueprintID, indicatorID)
	if err != nil {
		return fmt.Errorf("assessment.Repository.UnlinkIndicator: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessment.Repository.UnlinkIndicator: %w", ErrNotFound)
	}
	return nil
}

// ListBlueprintIndicators returns all performance indicators linked to a blueprint.
// This is used by the session/result service to validate that indicator IDs
// in a batch upsert belong to the session's blueprint.
func (r *PgRepository) ListBlueprintIndicators(ctx context.Context, blueprintID string) ([]LinkedIndicator, error) {
	const query = `
		SELECT pi.id, pi.description
		FROM assessment_blueprint_indicators abi
		JOIN performance_indicators pi ON pi.id = abi.indicator_id
		WHERE abi.blueprint_id = $1
		ORDER BY pi.sequence_order ASC
	`
	rows, err := r.pool.Query(ctx, query, blueprintID)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListBlueprintIndicators: %w", err)
	}
	defer rows.Close()

	var indicators []LinkedIndicator
	for rows.Next() {
		var li LinkedIndicator
		if err := rows.Scan(&li.ID, &li.Description); err != nil {
			return nil, fmt.Errorf("assessment.Repository.ListBlueprintIndicators: scan: %w", err)
		}
		indicators = append(indicators, li)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListBlueprintIndicators: rows: %w", err)
	}
	if indicators == nil {
		indicators = []LinkedIndicator{}
	}
	return indicators, nil
}

// IsIndicatorLinked checks if a specific indicator is already linked to a blueprint.
func (r *PgRepository) IsIndicatorLinked(ctx context.Context, blueprintID, indicatorID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM assessment_blueprint_indicators
			WHERE blueprint_id = $1 AND indicator_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, blueprintID, indicatorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("assessment.Repository.IsIndicatorLinked: %w", err)
	}
	return exists, nil
}

// ============================================================================
// ASSESSMENT SESSIONS
// ============================================================================

// CreateSession inserts a new assessment session and returns its ID.
func (r *PgRepository) CreateSession(ctx context.Context, s *AssessmentSession) (string, error) {
	const query = `
		INSERT INTO assessment_sessions
			(tenant_id, blueprint_id, class_id, assessed_by_user_id, date_administered)
		VALUES ($1, $2, $3, $4, $5::date)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		s.TenantID, s.BlueprintID, s.ClassID, s.AssessedByUserID, s.DateAdministered,
	).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return "", fmt.Errorf("assessment.Repository.CreateSession: %w", ErrNotFound)
		}
		return "", fmt.Errorf("assessment.Repository.CreateSession: %w", err)
	}
	return id, nil
}

// GetSessionByID retrieves a single session by primary key.
func (r *PgRepository) GetSessionByID(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
	const query = `
		SELECT id, tenant_id, blueprint_id, class_id, assessed_by_user_id,
		       date_administered::text, knec_upload_reference, created_at::text
		FROM assessment_sessions
		WHERE id = $1 AND tenant_id = $2
	`
	var s AssessmentSession
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&s.ID, &s.TenantID, &s.BlueprintID, &s.ClassID, &s.AssessedByUserID,
		&s.DateAdministered, &s.KNECUploadReference, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assessment.Repository.GetSessionByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("assessment.Repository.GetSessionByID: %w", err)
	}
	return &s, nil
}

// ListSessions returns sessions filtered by the given query parameters.
func (r *PgRepository) ListSessions(ctx context.Context, tenantID string, query ListSessionsQuery) ([]AssessmentSession, error) {
	baseQuery := `
		SELECT id, tenant_id, blueprint_id, class_id, assessed_by_user_id,
		       date_administered::text, knec_upload_reference, created_at::text
		FROM assessment_sessions
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if query.ClassID != "" {
		baseQuery += fmt.Sprintf(" AND class_id = $%d", argIdx)
		args = append(args, query.ClassID)
		argIdx++
	}
	if query.BlueprintID != "" {
		baseQuery += fmt.Sprintf(" AND blueprint_id = $%d", argIdx)
		args = append(args, query.BlueprintID)
	}

	baseQuery += " ORDER BY date_administered DESC, created_at DESC"

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListSessions: %w", err)
	}
	defer rows.Close()

	var sessions []AssessmentSession
	for rows.Next() {
		var s AssessmentSession
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.BlueprintID, &s.ClassID, &s.AssessedByUserID,
			&s.DateAdministered, &s.KNECUploadReference, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("assessment.Repository.ListSessions: scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListSessions: rows: %w", err)
	}
	if sessions == nil {
		sessions = []AssessmentSession{}
	}
	return sessions, nil
}

// UpdateSession applies changes to a session (date_administered, knec_upload_reference).
func (r *PgRepository) UpdateSession(ctx context.Context, s *AssessmentSession) error {
	const query = `
		UPDATE assessment_sessions
		SET date_administered = $1::date,
		    knec_upload_reference = $2
		WHERE id = $3 AND tenant_id = $4
	`
	tag, err := r.pool.Exec(ctx, query,
		s.DateAdministered, s.KNECUploadReference, s.ID, s.TenantID,
	)
	if err != nil {
		return fmt.Errorf("assessment.Repository.UpdateSession: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessment.Repository.UpdateSession: %w", ErrNotFound)
	}
	return nil
}

// DeleteSession removes a session. The ON DELETE CASCADE on learner_rubric_results
// handles cascading result deletion.
func (r *PgRepository) DeleteSession(ctx context.Context, id, tenantID string) error {
	const query = `
		DELETE FROM assessment_sessions
		WHERE id = $1 AND tenant_id = $2
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("assessment.Repository.DeleteSession: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessment.Repository.DeleteSession: %w", ErrNotFound)
	}
	return nil
}

// ============================================================================
// SESSION DETAIL (with results)
// ============================================================================

// GetSessionDetail retrieves a session with all its rubric results.
func (r *PgRepository) GetSessionDetail(ctx context.Context, id, tenantID string) (*SessionDetail, error) {
	// First fetch the session
	s, err := r.GetSessionByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.GetSessionDetail: %w", err)
	}

	// Then fetch all results for this session
	results, err := r.ListResults(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.GetSessionDetail: %w", err)
	}

	return &SessionDetail{
		AssessmentSession: *s,
		Results:           results,
	}, nil
}

// ============================================================================
// LEARNER RUBRIC RESULTS
// ============================================================================

// BatchUpsertResults inserts or updates rubric results in a single transaction.
// Returns the number of rows affected.
func (r *PgRepository) BatchUpsertResults(ctx context.Context, sessionID, tenantID string, results []LearnerRubricResult) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("assessment.Repository.BatchUpsertResults: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			// Log but don't mask the original error
			_ = err
		}
	}()

	const upsertQuery = `
		INSERT INTO learner_rubric_results
			(tenant_id, session_id, student_id, indicator_id, score_type, raw_score, rubric_level, teacher_observation_notes)
		VALUES ($1, $2, $3, $4, $5::lrr_score_type, $6, $7::cbc_rubric_level, $8)
		ON CONFLICT (session_id, student_id, indicator_id)
		DO UPDATE SET
			score_type               = EXCLUDED.score_type,
			raw_score                = EXCLUDED.raw_score,
			rubric_level             = EXCLUDED.rubric_level,
			teacher_observation_notes = EXCLUDED.teacher_observation_notes
	`

	rowsAffected := 0
	for _, r := range results {
		tag, err := tx.Exec(ctx, upsertQuery,
			tenantID, sessionID, r.StudentID, r.IndicatorID,
			r.ScoreType, r.RawScore, r.RubricLevel, r.TeacherObservationNotes,
		)
		if err != nil {
			if isFKViolation(err) {
				return 0, fmt.Errorf("assessment.Repository.BatchUpsertResults: fk violation: %w", ErrNotFound)
			}
			return 0, fmt.Errorf("assessment.Repository.BatchUpsertResults: upsert: %w", err)
		}
		rowsAffected += int(tag.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("assessment.Repository.BatchUpsertResults: commit: %w", err)
	}

	return rowsAffected, nil
}

// ListResults returns all rubric results for a given session.
func (r *PgRepository) ListResults(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
	const query = `
		SELECT id, session_id, student_id, indicator_id,
		       score_type::text, raw_score::text, rubric_level::text, teacher_observation_notes
		FROM learner_rubric_results
		WHERE session_id = $1 AND tenant_id = $2
		ORDER BY student_id, indicator_id
	`
	rows, err := r.pool.Query(ctx, query, sessionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListResults: %w", err)
	}
	defer rows.Close()

	var results []LearnerRubricResult
	for rows.Next() {
		var r LearnerRubricResult
		if err := rows.Scan(
			&r.ID, &r.SessionID, &r.StudentID, &r.IndicatorID,
			&r.ScoreType, &r.RawScore, &r.RubricLevel, &r.TeacherObservationNotes,
		); err != nil {
			return nil, fmt.Errorf("assessment.Repository.ListResults: scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListResults: rows: %w", err)
	}
	if results == nil {
		results = []LearnerRubricResult{}
	}
	return results, nil
}

// ============================================================================
// WEIGHT CONFIGS (read-only)
// ============================================================================

// ListWeightConfigs returns weight configs filtered by optional grade_level and/or target_exam.
func (r *PgRepository) ListWeightConfigs(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error) {
	baseQuery := `
		SELECT id, grade_level::text, assessment_type_code::text,
		       target_exam::text, weight_percent::text, effective_from
		FROM assessment_weight_configs
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if query.GradeLevel != "" {
		baseQuery += fmt.Sprintf(" AND grade_level = $%d::cbc_grade_level", argIdx)
		args = append(args, query.GradeLevel)
		argIdx++
	}
	if query.TargetExam != "" {
		baseQuery += fmt.Sprintf(" AND target_exam = $%d::knec_target_exam", argIdx)
		args = append(args, query.TargetExam)
	}

	baseQuery += " ORDER BY grade_level, assessment_type_code, effective_from DESC"

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListWeightConfigs: %w", err)
	}
	defer rows.Close()

	var configs []AssessmentWeightConfig
	for rows.Next() {
		var c AssessmentWeightConfig
		if err := rows.Scan(&c.ID, &c.GradeLevel, &c.AssessmentTypeCode,
			&c.TargetExam, &c.WeightPercent, &c.EffectiveFrom); err != nil {
			return nil, fmt.Errorf("assessment.Repository.ListWeightConfigs: scan: %w", err)
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assessment.Repository.ListWeightConfigs: rows: %w", err)
	}
	if configs == nil {
		configs = []AssessmentWeightConfig{}
	}
	return configs, nil
}
