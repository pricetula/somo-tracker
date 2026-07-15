package behavior

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository defines the contract for behavior persistence.
type Repository interface {
	// ── Categories ────────────────────────────────────────────────────────

	// ListCategories returns all categories for a school (including inactive).
	ListCategories(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error)

	// ListActiveCategories returns only active categories.
	ListActiveCategories(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error)

	// CreateCategory inserts a new behavior category.
	CreateCategory(ctx context.Context, tenantID, schoolID, name string, defaultSeverity *string) (*BehaviorCategory, error)

	// GetCategoryByID returns a single category by ID.
	GetCategoryByID(ctx context.Context, id, tenantID string) (*BehaviorCategory, error)

	// UpdateCategory updates a category's fields.
	UpdateCategory(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error)

	// ── Notes ─────────────────────────────────────────────────────────────

	// CreateNote inserts a new behavior note.
	CreateNote(ctx context.Context, tenantID, schoolID string, payload CreateNotePayload, authoredBy string) (*BehaviorNote, error)

	// GetPendingQueue returns all notes with PENDING_REVIEW status.
	GetPendingQueue(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error)

	// GetNoteByID returns a single behavior note by ID.
	GetNoteByID(ctx context.Context, id, tenantID string) (*BehaviorNote, error)

	// ReviewNote updates a note's status (approve/reject).
	ReviewNote(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error

	// GetNotesByStudentTerm returns approved notes for a student in a given term.
	GetNotesByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error)

	// UpdateNote updates the description of a behavior note.
	UpdateNote(ctx context.Context, id, tenantID string, description string) error
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed behavior repository.
func NewRepository(pools *database.Pools) Repository {
	return &pgRepository{pool: pools.PG}
}

// ── Categories ────────────────────────────────────────────────────────────

func (r *pgRepository) ListCategories(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
	query := `
		SELECT id, tenant_id, school_id, name, default_severity, is_active, created_at
		FROM behavior_categories
		WHERE tenant_id = $1 AND school_id = $2
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("behavior.Repository.ListCategories: %w", err)
	}
	defer rows.Close()

	var categories []BehaviorCategory
	for rows.Next() {
		var cat BehaviorCategory
		if err := rows.Scan(&cat.ID, &cat.TenantID, &cat.SchoolID, &cat.Name, &cat.DefaultSeverity, &cat.IsActive, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("behavior.Repository.ListCategories: scan: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

func (r *pgRepository) ListActiveCategories(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
	query := `
		SELECT id, tenant_id, school_id, name, default_severity, is_active, created_at
		FROM behavior_categories
		WHERE tenant_id = $1 AND school_id = $2 AND is_active = true
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("behavior.Repository.ListActiveCategories: %w", err)
	}
	defer rows.Close()

	var categories []BehaviorCategory
	for rows.Next() {
		var cat BehaviorCategory
		if err := rows.Scan(&cat.ID, &cat.TenantID, &cat.SchoolID, &cat.Name, &cat.DefaultSeverity, &cat.IsActive, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("behavior.Repository.ListActiveCategories: scan: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

func (r *pgRepository) CreateCategory(ctx context.Context, tenantID, schoolID, name string, defaultSeverity *string) (*BehaviorCategory, error) {
	var cat BehaviorCategory
	err := r.pool.QueryRow(ctx, `
		INSERT INTO behavior_categories (tenant_id, school_id, name, default_severity)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, school_id, name, default_severity, is_active, created_at
	`, tenantID, schoolID, name, defaultSeverity).Scan(
		&cat.ID, &cat.TenantID, &cat.SchoolID, &cat.Name, &cat.DefaultSeverity, &cat.IsActive, &cat.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("behavior.Repository.CreateCategory: %w", err)
	}
	return &cat, nil
}

func (r *pgRepository) GetCategoryByID(ctx context.Context, id, tenantID string) (*BehaviorCategory, error) {
	var cat BehaviorCategory
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, school_id, name, default_severity, is_active, created_at
		FROM behavior_categories
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&cat.ID, &cat.TenantID, &cat.SchoolID, &cat.Name, &cat.DefaultSeverity, &cat.IsActive, &cat.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("behavior.Repository.GetCategoryByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("behavior.Repository.GetCategoryByID: %w", err)
	}
	return &cat, nil
}

func (r *pgRepository) UpdateCategory(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error) {
	// Build dynamic UPDATE
	setClause := ""
	args := []interface{}{}
	argIdx := 1

	if payload.Name != nil {
		setClause += fmt.Sprintf("name = $%d, ", argIdx)
		args = append(args, *payload.Name)
		argIdx++
	}
	if payload.DefaultSeverity != nil {
		setClause += fmt.Sprintf("default_severity = $%d, ", argIdx)
		args = append(args, *payload.DefaultSeverity)
		argIdx++
	}
	if payload.IsActive != nil {
		setClause += fmt.Sprintf("is_active = $%d, ", argIdx)
		args = append(args, *payload.IsActive)
		argIdx++
	}

	if setClause == "" {
		return nil, fmt.Errorf("behavior.Repository.UpdateCategory: no fields to update: %w", ErrInvalidInput)
	}

	// Remove trailing ", "
	setClause = setClause[:len(setClause)-2]

	query := fmt.Sprintf(`
		UPDATE behavior_categories
		SET %s
		WHERE id = $%d AND tenant_id = $%d
		RETURNING id, tenant_id, school_id, name, default_severity, is_active, created_at
	`, setClause, argIdx, argIdx+1)

	args = append(args, id, tenantID)

	var cat BehaviorCategory
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&cat.ID, &cat.TenantID, &cat.SchoolID, &cat.Name, &cat.DefaultSeverity, &cat.IsActive, &cat.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("behavior.Repository.UpdateCategory: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("behavior.Repository.UpdateCategory: %w", err)
	}
	return &cat, nil
}

// ── Notes ─────────────────────────────────────────────────────────────────

func (r *pgRepository) CreateNote(ctx context.Context, tenantID, schoolID string, payload CreateNotePayload, authoredBy string) (*BehaviorNote, error) {
	var note BehaviorNote
	err := r.pool.QueryRow(ctx, `
		INSERT INTO behavior_notes
			(tenant_id, school_id, student_id, timetable_slot_id, date,
			 category_id, description, is_urgent, authored_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, school_id, student_id, timetable_slot_id, date,
		          category_id, description, is_urgent, status, authored_by_id,
		          reviewed_by_id, reviewed_at, created_at
	`, tenantID, schoolID, payload.StudentID, payload.TimetableSlotID, payload.Date,
		payload.CategoryID, payload.Description, payload.IsUrgent, authoredBy,
	).Scan(
		&note.ID, &note.TenantID, &note.SchoolID, &note.StudentID, &note.TimetableSlotID,
		&note.Date, &note.CategoryID, &note.Description, &note.IsUrgent, &note.Status,
		&note.AuthoredByID, &note.ReviewedByID, &note.ReviewedAt, &note.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("behavior.Repository.CreateNote: %w", err)
	}
	return &note, nil
}

func (r *pgRepository) GetPendingQueue(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error) {
	query := `
		SELECT
			bn.id,
			bn.student_id,
			s.full_name AS student_full_name,
			c.grade_level || ' ' || COALESCE(str.name, '') AS class_name,
			bn.category_id,
			bc.name AS category_name,
			bn.description,
			bn.is_urgent,
			bn.authored_by_id,
			u.full_name AS authored_by_name,
			bn.date,
			bn.status
		FROM behavior_notes bn
		JOIN cbc_students s ON s.id = bn.student_id AND s.tenant_id = bn.tenant_id
		JOIN cbc_timetable_slots ts ON ts.id = bn.timetable_slot_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = bn.tenant_id
		LEFT JOIN cbc_streams str ON str.id = c.stream_id
		JOIN behavior_categories bc ON bc.id = bn.category_id
		JOIN users u ON u.id = bn.authored_by_id AND u.tenant_id = bn.tenant_id
		WHERE bn.tenant_id = $1 AND bn.school_id = $2 AND bn.status = 'PENDING_REVIEW'
		ORDER BY bn.is_urgent DESC, bn.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("behavior.Repository.GetPendingQueue: %w", err)
	}
	defer rows.Close()

	var notes []PendingNoteItem
	for rows.Next() {
		var item PendingNoteItem
		if err := rows.Scan(
			&item.ID, &item.StudentID, &item.StudentFullName, &item.ClassName,
			&item.CategoryID, &item.CategoryName, &item.Description, &item.IsUrgent,
			&item.AuthoredByID, &item.AuthoredByName, &item.Date, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("behavior.Repository.GetPendingQueue: scan: %w", err)
		}
		notes = append(notes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("behavior.Repository.GetPendingQueue: rows: %w", err)
	}

	return &PendingNotesResponse{Notes: notes}, nil
}

func (r *pgRepository) GetNoteByID(ctx context.Context, id, tenantID string) (*BehaviorNote, error) {
	var note BehaviorNote
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, school_id, student_id, timetable_slot_id, date,
		       category_id, description, is_urgent, status, authored_by_id,
		       reviewed_by_id, reviewed_at, created_at
		FROM behavior_notes
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&note.ID, &note.TenantID, &note.SchoolID, &note.StudentID, &note.TimetableSlotID,
		&note.Date, &note.CategoryID, &note.Description, &note.IsUrgent, &note.Status,
		&note.AuthoredByID, &note.ReviewedByID, &note.ReviewedAt, &note.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("behavior.Repository.GetNoteByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("behavior.Repository.GetNoteByID: %w", err)
	}
	return &note, nil
}

func (r *pgRepository) ReviewNote(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error {
	var status BehaviorNoteStatus
	switch decision.Decision {
	case "APPROVED":
		status = StatusApproved
	case "REJECTED":
		status = StatusRejected
	default:
		return fmt.Errorf("behavior.Repository.ReviewNote: invalid decision %q: %w", decision.Decision, ErrInvalidInput)
	}

	now := time.Now()
	result, err := r.pool.Exec(ctx, `
		UPDATE behavior_notes
		SET status = $1, reviewed_by_id = $2, reviewed_at = $3
		WHERE id = $4 AND tenant_id = $5 AND status = 'PENDING_REVIEW'
	`, status, reviewedBy, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("behavior.Repository.ReviewNote: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("behavior.Repository.ReviewNote: %w", ErrNotFound)
	}
	return nil
}

func (r *pgRepository) GetNotesByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error) {
	query := `
		SELECT
			bn.id,
			bn.student_id,
			s.full_name AS student_full_name,
			c.grade_level || ' ' || COALESCE(str.name, '') AS class_name,
			bn.category_id,
			bc.name AS category_name,
			bn.description,
			bn.is_urgent,
			bn.authored_by_id,
			u.full_name AS authored_by_name,
			bn.date,
			bn.status
		FROM behavior_notes bn
		JOIN cbc_students s ON s.id = bn.student_id AND s.tenant_id = bn.tenant_id
		JOIN cbc_timetable_slots ts ON ts.id = bn.timetable_slot_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = bn.tenant_id
		LEFT JOIN cbc_streams str ON str.id = c.stream_id
		JOIN behavior_categories bc ON bc.id = bn.category_id
		JOIN users u ON u.id = bn.authored_by_id AND u.tenant_id = bn.tenant_id
		JOIN attendance_records ar ON ar.timetable_slot_id = bn.timetable_slot_id
			AND ar.date = bn.date AND ar.student_id = bn.student_id
		JOIN academic_terms at ON at.id = ar.academic_term_id
		WHERE bn.tenant_id = $1
		  AND bn.school_id = $2
		  AND bn.student_id = $3
		  AND at.id = $4
		  AND bn.status IN ('APPROVED', 'INCLUDED_IN_REPORT')
		ORDER BY bn.date DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("behavior.Repository.GetNotesByStudentTerm: %w", err)
	}
	defer rows.Close()

	var notes []PendingNoteItem
	for rows.Next() {
		var item PendingNoteItem
		if err := rows.Scan(
			&item.ID, &item.StudentID, &item.StudentFullName, &item.ClassName,
			&item.CategoryID, &item.CategoryName, &item.Description, &item.IsUrgent,
			&item.AuthoredByID, &item.AuthoredByName, &item.Date, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("behavior.Repository.GetNotesByStudentTerm: scan: %w", err)
		}
		notes = append(notes, item)
	}
	return notes, rows.Err()
}

func (r *pgRepository) UpdateNote(ctx context.Context, id, tenantID string, description string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE behavior_notes
		SET description = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, description, id, tenantID)
	if err != nil {
		return fmt.Errorf("behavior.Repository.UpdateNote: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("behavior.Repository.UpdateNote: %w", ErrNotFound)
	}
	return nil
}

// Ensure compile-time check that *pgRepository satisfies Repository.
var _ Repository = (*pgRepository)(nil)
