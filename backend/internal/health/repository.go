package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles health database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ═══════════════════════════════════════════════════════════════════════════
// MEDICAL INCIDENTS
// ═══════════════════════════════════════════════════════════════════════════

// CreateIncident inserts a new medical incident record.
func (r *PgRepository) CreateIncident(ctx context.Context, params CreateIncidentParams) (string, error) {
	const query = `
		INSERT INTO medical_incidents (tenant_id, student_id, incident_timestamp, symptoms, action_taken, logged_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.StudentID,
		params.IncidentTimestamp,
		params.Symptoms,
		params.ActionTaken,
		params.LoggedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("health.Repository.CreateIncident: %w", err)
	}
	return id, nil
}

// GetIncidentByID retrieves a medical incident by ID.
func (r *PgRepository) GetIncidentByID(ctx context.Context, id, tenantID string) (*MedicalIncident, error) {
	const query = `
		SELECT mi.id, mi.tenant_id, mi.student_id, mi.incident_timestamp,
		       mi.symptoms, mi.action_taken, mi.logged_by, u.full_name AS logged_by_name,
		       mi.created_at, mi.updated_at
		FROM medical_incidents mi
		JOIN users u ON u.id = mi.logged_by
		WHERE mi.id = $1 AND mi.tenant_id = $2
	`
	var inc MedicalIncident
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&inc.ID, &inc.TenantID, &inc.StudentID, &inc.IncidentTimestamp,
		&inc.Symptoms, &inc.ActionTaken, &inc.LoggedBy, &inc.LoggedByName,
		&inc.CreatedAt, &inc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("health.Repository.GetIncidentByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("health.Repository.GetIncidentByID: %w", err)
	}
	return &inc, nil
}

// ListIncidentsByStudent returns all incidents for a student, ordered by most recent first.
func (r *PgRepository) ListIncidentsByStudent(ctx context.Context, studentID, tenantID string) ([]MedicalIncident, error) {
	const query = `
		SELECT mi.id, mi.tenant_id, mi.student_id, mi.incident_timestamp,
		       mi.symptoms, mi.action_taken, mi.logged_by, u.full_name AS logged_by_name,
		       mi.created_at, mi.updated_at
		FROM medical_incidents mi
		JOIN users u ON u.id = mi.logged_by
		WHERE mi.student_id = $1 AND mi.tenant_id = $2
		ORDER BY mi.incident_timestamp DESC
	`
	rows, err := r.pool.Query(ctx, query, studentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("health.Repository.ListIncidentsByStudent: %w", err)
	}
	defer rows.Close()

	var items []MedicalIncident
	for rows.Next() {
		var inc MedicalIncident
		if err := rows.Scan(
			&inc.ID, &inc.TenantID, &inc.StudentID, &inc.IncidentTimestamp,
			&inc.Symptoms, &inc.ActionTaken, &inc.LoggedBy, &inc.LoggedByName,
			&inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("health.Repository.ScanIncident: %w", err)
		}
		items = append(items, inc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("health.Repository.Rows: %w", err)
	}
	if items == nil {
		items = []MedicalIncident{}
	}
	return items, nil
}

// ListIncidentsBySchool returns paginated incidents for a school.
func (r *PgRepository) ListIncidentsBySchool(ctx context.Context, tenantID, schoolID string, limit, offset int) ([]MedicalIncident, int, error) {
	// Count
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM medical_incidents mi
		JOIN cbc_students s ON s.id = mi.student_id
		WHERE mi.tenant_id = $1 AND s.school_id = $2
	`, tenantID, schoolID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("health.Repository.CountIncidents: %w", err)
	}

	// Fetch
	rows, err := r.pool.Query(ctx, `
		SELECT mi.id, mi.tenant_id, mi.student_id, mi.incident_timestamp,
		       mi.symptoms, mi.action_taken, mi.logged_by, u.full_name AS logged_by_name,
		       mi.created_at, mi.updated_at
		FROM medical_incidents mi
		JOIN cbc_students s ON s.id = mi.student_id
		JOIN users u ON u.id = mi.logged_by
		WHERE mi.tenant_id = $1 AND s.school_id = $2
		ORDER BY mi.incident_timestamp DESC
		LIMIT $3 OFFSET $4
	`, tenantID, schoolID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("health.Repository.ListIncidentsBySchool: %w", err)
	}
	defer rows.Close()

	var items []MedicalIncident
	for rows.Next() {
		var inc MedicalIncident
		if err := rows.Scan(
			&inc.ID, &inc.TenantID, &inc.StudentID, &inc.IncidentTimestamp,
			&inc.Symptoms, &inc.ActionTaken, &inc.LoggedBy, &inc.LoggedByName,
			&inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("health.Repository.ScanIncident: %w", err)
		}
		items = append(items, inc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("health.Repository.Rows: %w", err)
	}
	if items == nil {
		items = []MedicalIncident{}
	}
	return items, total, nil
}

// UpdateIncident updates a medical incident.
func (r *PgRepository) UpdateIncident(ctx context.Context, id, tenantID string, payload UpdateMedicalIncidentPayload) error {
	const query = `
		UPDATE medical_incidents
		SET symptoms   = COALESCE($3, symptoms),
		    action_taken = COALESCE($4, action_taken)
		WHERE id = $1 AND tenant_id = $2
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID, payload.Symptoms, payload.ActionTaken)
	if err != nil {
		return fmt.Errorf("health.Repository.UpdateIncident: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("health.Repository.UpdateIncident: %w", ErrNotFound)
	}
	return nil
}

// DeleteIncident deletes a medical incident.
func (r *PgRepository) DeleteIncident(ctx context.Context, id, tenantID string) error {
	const query = `DELETE FROM medical_incidents WHERE id = $1 AND tenant_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("health.Repository.DeleteIncident: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("health.Repository.DeleteIncident: %w", ErrNotFound)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// STUDENT HEALTH PROFILES
// ═══════════════════════════════════════════════════════════════════════════

// UpsertProfile creates or updates a student health profile.
func (r *PgRepository) UpsertProfile(ctx context.Context, params UpsertProfileParams) (*StudentHealthProfile, error) {
	const query = `
		INSERT INTO student_health_profiles (tenant_id, student_id, blood_group, allergies, chronic_conditions, emergency_instructions)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (student_id) DO UPDATE SET
			blood_group            = COALESCE($3, student_health_profiles.blood_group),
			allergies              = CASE WHEN $4 IS NOT NULL THEN $4 ELSE student_health_profiles.allergies END,
			chronic_conditions     = CASE WHEN $5 IS NOT NULL THEN $5 ELSE student_health_profiles.chronic_conditions END,
			emergency_instructions = COALESCE($6, student_health_profiles.emergency_instructions)
		RETURNING id, tenant_id, student_id, blood_group, allergies, chronic_conditions, emergency_instructions, created_at, updated_at
	`
	var profile StudentHealthProfile
	err := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.StudentID,
		params.BloodGroup,
		params.Allergies,
		params.ChronicConditions,
		params.EmergencyInstructions,
	).Scan(
		&profile.ID, &profile.TenantID, &profile.StudentID,
		&profile.BloodGroup, &profile.Allergies, &profile.ChronicConditions,
		&profile.EmergencyInstructions, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("health.Repository.UpsertProfile: %w", err)
	}
	return &profile, nil
}

// GetProfileByStudent retrieves a health profile by student ID.
func (r *PgRepository) GetProfileByStudent(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error) {
	const query = `
		SELECT id, tenant_id, student_id, blood_group, allergies, chronic_conditions, emergency_instructions, created_at, updated_at
		FROM student_health_profiles
		WHERE student_id = $1 AND tenant_id = $2
	`
	var profile StudentHealthProfile
	err := r.pool.QueryRow(ctx, query, studentID, tenantID).Scan(
		&profile.ID, &profile.TenantID, &profile.StudentID,
		&profile.BloodGroup, &profile.Allergies, &profile.ChronicConditions,
		&profile.EmergencyInstructions, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("health.Repository.GetProfileByStudent: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("health.Repository.GetProfileByStudent: %w", err)
	}
	return &profile, nil
}

// GetProfileByID retrieves a health profile by its ID.
func (r *PgRepository) GetProfileByID(ctx context.Context, id, tenantID string) (*StudentHealthProfile, error) {
	const query = `
		SELECT id, tenant_id, student_id, blood_group, allergies, chronic_conditions, emergency_instructions, created_at, updated_at
		FROM student_health_profiles
		WHERE id = $1 AND tenant_id = $2
	`
	var profile StudentHealthProfile
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&profile.ID, &profile.TenantID, &profile.StudentID,
		&profile.BloodGroup, &profile.Allergies, &profile.ChronicConditions,
		&profile.EmergencyInstructions, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("health.Repository.GetProfileByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("health.Repository.GetProfileByID: %w", err)
	}
	return &profile, nil
}

// compile-time interface check
var _ Repository = (*PgRepository)(nil)
