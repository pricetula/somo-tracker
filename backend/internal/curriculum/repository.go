package curriculum

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles curriculum database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ── Helpers ───────────────────────────────────────────────────────────────

// isFKViolation checks if an error is a PostgreSQL foreign key violation (23503).
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	result := clauses[0]
	for _, c := range clauses[1:] {
		result += sep + c
	}
	return result
}

// ── Learning Areas ────────────────────────────────────────────────────────

// CreateLearningArea inserts a new cbc_learning_area and returns its ID.
func (r *PgRepository) CreateLearningArea(ctx context.Context, params CreateLearningAreaParams) (string, error) {
	const query = `
		INSERT INTO cbc_learning_areas (tenant_id, school_id, name, code, education_level)
		VALUES ($1, $2, $3, $4, $5::cbc_education_level)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.SchoolID,
		params.Name,
		params.Code,
		params.EducationLevel,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("curriculum.Repository.CreateLearningArea: %w", err)
	}
	return id, nil
}

// GetLearningAreaByID retrieves a single learning area by ID, scoped to tenant + school.
func (r *PgRepository) GetLearningAreaByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
	const query = `
		SELECT id, tenant_id, school_id, name, code, education_level::text
		FROM cbc_learning_areas
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var la LearningArea
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).
		Scan(&la.ID, &la.TenantID, &la.SchoolID, &la.Name, &la.Code, &la.EducationLevel)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("curriculum.Repository.GetLearningAreaByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("curriculum.Repository.GetLearningAreaByID: %w", err)
	}
	return &la, nil
}

// ListLearningAreas returns all learning areas for the given tenant and school,
// optionally filtered by education_level.
func (r *PgRepository) ListLearningAreas(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
	query := `
		SELECT id, tenant_id, school_id, name, code, education_level::text
		FROM cbc_learning_areas
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []interface{}{tenantID, schoolID}

	if educationLevel != nil && *educationLevel != "" {
		args = append(args, *educationLevel)
		query += fmt.Sprintf(" AND education_level = $%d::cbc_education_level", len(args))
	}

	query += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListLearningAreas: %w", err)
	}
	defer rows.Close()

	var areas []LearningArea
	for rows.Next() {
		var la LearningArea
		if err := rows.Scan(&la.ID, &la.TenantID, &la.SchoolID, &la.Name, &la.Code, &la.EducationLevel); err != nil {
			return nil, fmt.Errorf("curriculum.Repository.ListLearningAreas: scan: %w", err)
		}
		areas = append(areas, la)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListLearningAreas: rows: %w", err)
	}

	if areas == nil {
		areas = []LearningArea{}
	}

	return areas, nil
}

// UpdateLearningArea modifies learning area fields. Only non-nil fields are applied.
func (r *PgRepository) UpdateLearningArea(ctx context.Context, params UpdateLearningAreaParams) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *params.Name)
		argIdx++
	}
	if params.Code != nil {
		setClauses = append(setClauses, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *params.Code)
		argIdx++
	}
	if params.EducationLevel != nil {
		setClauses = append(setClauses, fmt.Sprintf("education_level = $%d::cbc_education_level", argIdx))
		args = append(args, *params.EducationLevel)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("curriculum.Repository.UpdateLearningArea: %w", ErrInvalidInput)
	}

	args = append(args, params.ID, params.TenantID, params.SchoolID)
	query := fmt.Sprintf(`
		UPDATE cbc_learning_areas
		SET %s
		WHERE id = $%d AND tenant_id = $%d AND school_id = $%d
	`, joinClauses(setClauses, ", "), argIdx, argIdx+1, argIdx+2)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("curriculum.Repository.UpdateLearningArea: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.UpdateLearningArea: %w", ErrNotFound)
	}

	return nil
}

// DeleteLearningArea removes a learning area by ID, scoped to tenant + school.
func (r *PgRepository) DeleteLearningArea(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		DELETE FROM cbc_learning_areas
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		if isFKViolation(err) {
			return fmt.Errorf("curriculum.Repository.DeleteLearningArea: %w", ErrReferenceProtected)
		}
		return fmt.Errorf("curriculum.Repository.DeleteLearningArea: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.DeleteLearningArea: %w", ErrNotFound)
	}
	return nil
}

// ── Strands ───────────────────────────────────────────────────────────────

// CreateStrand inserts a new cbc_strand and returns its ID.
func (r *PgRepository) CreateStrand(ctx context.Context, params CreateStrandParams) (string, error) {
	const query = `
		INSERT INTO cbc_strands (learning_area_id, name)
		VALUES ($1, $2)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, params.LearningAreaID, params.Name).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return "", fmt.Errorf("curriculum.Repository.CreateStrand: %w", ErrNotFound)
		}
		return "", fmt.Errorf("curriculum.Repository.CreateStrand: %w", err)
	}
	return id, nil
}

// GetStrandByID retrieves a single strand by ID.
func (r *PgRepository) GetStrandByID(ctx context.Context, id string) (*Strand, error) {
	const query = `
		SELECT id, learning_area_id, name
		FROM cbc_strands
		WHERE id = $1
	`
	var s Strand
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.LearningAreaID, &s.Name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("curriculum.Repository.GetStrandByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("curriculum.Repository.GetStrandByID: %w", err)
	}
	return &s, nil
}

// ListStrandsByLearningArea returns all strands for a given learning area.
func (r *PgRepository) ListStrandsByLearningArea(ctx context.Context, learningAreaID string) ([]Strand, error) {
	const query = `
		SELECT id, learning_area_id, name
		FROM cbc_strands
		WHERE learning_area_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListStrandsByLearningArea: %w", err)
	}
	defer rows.Close()

	var strands []Strand
	for rows.Next() {
		var s Strand
		if err := rows.Scan(&s.ID, &s.LearningAreaID, &s.Name); err != nil {
			return nil, fmt.Errorf("curriculum.Repository.ListStrandsByLearningArea: scan: %w", err)
		}
		strands = append(strands, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListStrandsByLearningArea: rows: %w", err)
	}

	if strands == nil {
		strands = []Strand{}
	}

	return strands, nil
}

// UpdateStrand modifies a strand's name.
func (r *PgRepository) UpdateStrand(ctx context.Context, params UpdateStrandParams) error {
	if params.Name == nil {
		return fmt.Errorf("curriculum.Repository.UpdateStrand: %w", ErrInvalidInput)
	}

	const query = `
		UPDATE cbc_strands
		SET name = $1
		WHERE id = $2
	`
	result, err := r.pool.Exec(ctx, query, *params.Name, params.ID)
	if err != nil {
		return fmt.Errorf("curriculum.Repository.UpdateStrand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.UpdateStrand: %w", ErrNotFound)
	}
	return nil
}

// DeleteStrand removes a strand by ID.
func (r *PgRepository) DeleteStrand(ctx context.Context, id string) error {
	const query = `DELETE FROM cbc_strands WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		if isFKViolation(err) {
			return fmt.Errorf("curriculum.Repository.DeleteStrand: %w", ErrReferenceProtected)
		}
		return fmt.Errorf("curriculum.Repository.DeleteStrand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.DeleteStrand: %w", ErrNotFound)
	}
	return nil
}

// ── Sub-Strands ───────────────────────────────────────────────────────────

// CreateSubStrand inserts a new cbc_sub_strand and returns its ID.
func (r *PgRepository) CreateSubStrand(ctx context.Context, params CreateSubStrandParams) (string, error) {
	const query = `
		INSERT INTO cbc_sub_strands (strand_id, name)
		VALUES ($1, $2)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, params.StrandID, params.Name).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return "", fmt.Errorf("curriculum.Repository.CreateSubStrand: %w", ErrNotFound)
		}
		return "", fmt.Errorf("curriculum.Repository.CreateSubStrand: %w", err)
	}
	return id, nil
}

// GetSubStrandByID retrieves a single sub-strand by ID.
func (r *PgRepository) GetSubStrandByID(ctx context.Context, id string) (*SubStrand, error) {
	const query = `
		SELECT id, strand_id, name
		FROM cbc_sub_strands
		WHERE id = $1
	`
	var s SubStrand
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.StrandID, &s.Name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("curriculum.Repository.GetSubStrandByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("curriculum.Repository.GetSubStrandByID: %w", err)
	}
	return &s, nil
}

// ListSubStrandsByStrand returns all sub-strands for a given strand.
func (r *PgRepository) ListSubStrandsByStrand(ctx context.Context, strandID string) ([]SubStrand, error) {
	const query = `
		SELECT id, strand_id, name
		FROM cbc_sub_strands
		WHERE strand_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, strandID)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListSubStrandsByStrand: %w", err)
	}
	defer rows.Close()

	var subs []SubStrand
	for rows.Next() {
		var s SubStrand
		if err := rows.Scan(&s.ID, &s.StrandID, &s.Name); err != nil {
			return nil, fmt.Errorf("curriculum.Repository.ListSubStrandsByStrand: scan: %w", err)
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListSubStrandsByStrand: rows: %w", err)
	}

	if subs == nil {
		subs = []SubStrand{}
	}

	return subs, nil
}

// UpdateSubStrand modifies a sub-strand's name.
func (r *PgRepository) UpdateSubStrand(ctx context.Context, params UpdateSubStrandParams) error {
	if params.Name == nil {
		return fmt.Errorf("curriculum.Repository.UpdateSubStrand: %w", ErrInvalidInput)
	}

	const query = `
		UPDATE cbc_sub_strands
		SET name = $1
		WHERE id = $2
	`
	result, err := r.pool.Exec(ctx, query, *params.Name, params.ID)
	if err != nil {
		return fmt.Errorf("curriculum.Repository.UpdateSubStrand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.UpdateSubStrand: %w", ErrNotFound)
	}
	return nil
}

// DeleteSubStrand removes a sub-strand by ID.
func (r *PgRepository) DeleteSubStrand(ctx context.Context, id string) error {
	const query = `DELETE FROM cbc_sub_strands WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		if isFKViolation(err) {
			return fmt.Errorf("curriculum.Repository.DeleteSubStrand: %w", ErrReferenceProtected)
		}
		return fmt.Errorf("curriculum.Repository.DeleteSubStrand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.DeleteSubStrand: %w", ErrNotFound)
	}
	return nil
}

// ── Performance Indicators ────────────────────────────────────────────────

// CreatePerformanceIndicator inserts a new performance_indicator and returns its ID.
func (r *PgRepository) CreatePerformanceIndicator(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error) {
	const query = `
		INSERT INTO performance_indicators (sub_strand_id, description, sequence_order)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, params.SubStrandID, params.Description, *params.SequenceOrder).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return "", fmt.Errorf("curriculum.Repository.CreatePerformanceIndicator: %w", ErrNotFound)
		}
		return "", fmt.Errorf("curriculum.Repository.CreatePerformanceIndicator: %w", err)
	}
	return id, nil
}

// GetPerformanceIndicatorByID retrieves a single performance indicator by ID.
func (r *PgRepository) GetPerformanceIndicatorByID(ctx context.Context, id string) (*PerformanceIndicator, error) {
	const query = `
		SELECT id, sub_strand_id, description, sequence_order
		FROM performance_indicators
		WHERE id = $1
	`
	var pi PerformanceIndicator
	err := r.pool.QueryRow(ctx, query, id).Scan(&pi.ID, &pi.SubStrandID, &pi.Description, &pi.SequenceOrder)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("curriculum.Repository.GetPerformanceIndicatorByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("curriculum.Repository.GetPerformanceIndicatorByID: %w", err)
	}
	return &pi, nil
}

// ListPerformanceIndicatorsBySubStrand returns all performance indicators for a given sub-strand,
// ordered by sequence_order ascending.
func (r *PgRepository) ListPerformanceIndicatorsBySubStrand(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error) {
	const query = `
		SELECT id, sub_strand_id, description, sequence_order
		FROM performance_indicators
		WHERE sub_strand_id = $1
		ORDER BY sequence_order ASC
	`
	rows, err := r.pool.Query(ctx, query, subStrandID)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListPerformanceIndicatorsBySubStrand: %w", err)
	}
	defer rows.Close()

	var indicators []PerformanceIndicator
	for rows.Next() {
		var pi PerformanceIndicator
		if err := rows.Scan(&pi.ID, &pi.SubStrandID, &pi.Description, &pi.SequenceOrder); err != nil {
			return nil, fmt.Errorf("curriculum.Repository.ListPerformanceIndicatorsBySubStrand: scan: %w", err)
		}
		indicators = append(indicators, pi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("curriculum.Repository.ListPerformanceIndicatorsBySubStrand: rows: %w", err)
	}

	if indicators == nil {
		indicators = []PerformanceIndicator{}
	}

	return indicators, nil
}

// UpdatePerformanceIndicator modifies a performance indicator's fields.
func (r *PgRepository) UpdatePerformanceIndicator(ctx context.Context, params UpdatePerformanceIndicatorParams) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *params.Description)
		argIdx++
	}
	if params.SequenceOrder != nil {
		setClauses = append(setClauses, fmt.Sprintf("sequence_order = $%d", argIdx))
		args = append(args, *params.SequenceOrder)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("curriculum.Repository.UpdatePerformanceIndicator: %w", ErrInvalidInput)
	}

	args = append(args, params.ID)
	query := fmt.Sprintf(`
		UPDATE performance_indicators
		SET %s
		WHERE id = $%d
	`, joinClauses(setClauses, ", "), argIdx)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("curriculum.Repository.UpdatePerformanceIndicator: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.UpdatePerformanceIndicator: %w", ErrNotFound)
	}
	return nil
}

// DeletePerformanceIndicator removes a performance indicator by ID.
func (r *PgRepository) DeletePerformanceIndicator(ctx context.Context, id string) error {
	const query = `DELETE FROM performance_indicators WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		if isFKViolation(err) {
			return fmt.Errorf("curriculum.Repository.DeletePerformanceIndicator: %w", ErrReferenceProtected)
		}
		return fmt.Errorf("curriculum.Repository.DeletePerformanceIndicator: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.DeletePerformanceIndicator: %w", ErrNotFound)
	}
	return nil
}

// GetMaxSequenceOrder returns the highest sequence_order for a given sub_strand.
// Returns 0 if no indicators exist.
func (r *PgRepository) GetMaxSequenceOrder(ctx context.Context, subStrandID string) (int, error) {
	const query = `
		SELECT COALESCE(MAX(sequence_order), 0)
		FROM performance_indicators
		WHERE sub_strand_id = $1
	`
	var max int
	err := r.pool.QueryRow(ctx, query, subStrandID).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("curriculum.Repository.GetMaxSequenceOrder: %w", err)
	}
	return max, nil
}

// ── Tree ──────────────────────────────────────────────────────────────────

// GetTree fetches a learning area with its full hierarchy: strands → sub-strands → indicators.
func (r *PgRepository) GetTree(ctx context.Context, learningAreaID string) (*LearningAreaTree, error) {
	// 1. Fetch the learning area (without tenant/school filter — caller must validate)
	const laQuery = `
		SELECT id, tenant_id, school_id, name, code, education_level::text
		FROM cbc_learning_areas
		WHERE id = $1
	`
	var la LearningArea
	err := r.pool.QueryRow(ctx, laQuery, learningAreaID).
		Scan(&la.ID, &la.TenantID, &la.SchoolID, &la.Name, &la.Code, &la.EducationLevel)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("curriculum.Repository.GetTree: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("curriculum.Repository.GetTree: %w", err)
	}

	// 2. Fetch strands
	strands, err := r.ListStrandsByLearningArea(ctx, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Repository.GetTree: %w", err)
	}

	// 3. Build strand trees with nested sub-strands and indicators
	tree := &LearningAreaTree{
		LearningArea: la,
		Strands:      make([]StrandTree, 0, len(strands)),
	}

	for _, strand := range strands {
		subStrands, err := r.ListSubStrandsByStrand(ctx, strand.ID)
		if err != nil {
			return nil, fmt.Errorf("curriculum.Repository.GetTree: %w", err)
		}

		st := StrandTree{
			Strand:     strand,
			SubStrands: make([]SubStrandTree, 0, len(subStrands)),
		}

		for _, sub := range subStrands {
			indicators, err := r.ListPerformanceIndicatorsBySubStrand(ctx, sub.ID)
			if err != nil {
				return nil, fmt.Errorf("curriculum.Repository.GetTree: %w", err)
			}

			st.SubStrands = append(st.SubStrands, SubStrandTree{
				SubStrand:             sub,
				PerformanceIndicators: indicators,
			})
		}

		tree.Strands = append(tree.Strands, st)
	}

	return tree, nil
}

// ── Cross-Domain Helpers ────────────────────────────────────────────────────

// GetPerformanceIndicatorEducationLevel traverses the hierarchy from a
// performance indicator up to its learning area and returns the education level.
func (r *PgRepository) GetPerformanceIndicatorEducationLevel(ctx context.Context, indicatorID string) (string, error) {
	const query = `
		SELECT la.education_level::text
		FROM performance_indicators pi
		JOIN cbc_sub_strands ss ON ss.id = pi.sub_strand_id
		JOIN cbc_strands s ON s.id = ss.strand_id
		JOIN cbc_learning_areas la ON la.id = s.learning_area_id
		WHERE pi.id = $1
	`
	var educationLevel string
	err := r.pool.QueryRow(ctx, query, indicatorID).Scan(&educationLevel)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("curriculum.Repository.GetPerformanceIndicatorEducationLevel: %w", ErrNotFound)
		}
		return "", fmt.Errorf("curriculum.Repository.GetPerformanceIndicatorEducationLevel: %w", err)
	}
	return educationLevel, nil
}

// ── Parent-Validation Helpers ─────────────────────────────────────────────

// VerifyLearningAreaBelongsToTenant checks that a learning area exists for the given tenant + school.
func (r *PgRepository) VerifyLearningAreaBelongsToTenant(ctx context.Context, id, tenantID, schoolID string) error {
	_, err := r.GetLearningAreaByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return err
	}
	return nil
}

// VerifyStrandInTenantSchool checks that a strand's learning area belongs to the given tenant + school.
// Returns the learning_area_id on success.
func (r *PgRepository) VerifyStrandInTenantSchool(ctx context.Context, strandID, tenantID, schoolID string) (string, error) {
	const query = `
		SELECT cs.learning_area_id
		FROM cbc_strands cs
		JOIN cbc_learning_areas cla ON cla.id = cs.learning_area_id
		WHERE cs.id = $1 AND cla.tenant_id = $2 AND cla.school_id = $3
	`
	var learningAreaID string
	err := r.pool.QueryRow(ctx, query, strandID, tenantID, schoolID).Scan(&learningAreaID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("curriculum.Repository.VerifyStrandInTenantSchool: %w", ErrNotFound)
		}
		return "", fmt.Errorf("curriculum.Repository.VerifyStrandInTenantSchool: %w", err)
	}
	return learningAreaID, nil
}

// VerifySubStrandInTenantSchool checks that a sub-strand's strand → learning area chain
// belongs to the given tenant + school. Returns the strand_id on success.
func (r *PgRepository) VerifySubStrandInTenantSchool(ctx context.Context, subStrandID, tenantID, schoolID string) (string, error) {
	const query = `
		SELECT css.strand_id
		FROM cbc_sub_strands css
		JOIN cbc_strands cs ON cs.id = css.strand_id
		JOIN cbc_learning_areas cla ON cla.id = cs.learning_area_id
		WHERE css.id = $1 AND cla.tenant_id = $2 AND cla.school_id = $3
	`
	var strandID string
	err := r.pool.QueryRow(ctx, query, subStrandID, tenantID, schoolID).Scan(&strandID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("curriculum.Repository.VerifySubStrandInTenantSchool: %w", ErrNotFound)
		}
		return "", fmt.Errorf("curriculum.Repository.VerifySubStrandInTenantSchool: %w", err)
	}
	return strandID, nil
}
