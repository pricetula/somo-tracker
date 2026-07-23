// Package health manages student health records and medical incidents.
// Includes student_health_profiles (blood group, allergies, conditions) and
// medical_incidents (per-occurrence records logged by nurses).
package health

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound      = fmt.Errorf("health record not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("health record already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid health input: %w", middleware.ErrInvalidInput)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
)

// ── Medical Incident ─────────────────────────────────────────────────────

// MedicalIncident represents a medical incident logged by a nurse.
type MedicalIncident struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"-"`
	StudentID         string    `json:"student_id"`
	IncidentTimestamp time.Time `json:"incident_timestamp"`
	Symptoms          string    `json:"symptoms"`
	ActionTaken       string    `json:"action_taken"`
	LoggedBy          string    `json:"logged_by"`
	LoggedByName      string    `json:"logged_by_name,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CreateMedicalIncidentPayload is the request body for logging an incident.
type CreateMedicalIncidentPayload struct {
	StudentID         string `json:"student_id"`
	IncidentTimestamp string `json:"incident_timestamp"`
	Symptoms          string `json:"symptoms"`
	ActionTaken       string `json:"action_taken"`
}

// UpdateMedicalIncidentPayload is the request body for updating an incident.
type UpdateMedicalIncidentPayload struct {
	Symptoms    *string `json:"symptoms,omitempty"`
	ActionTaken *string `json:"action_taken,omitempty"`
}

// ── Student Health Profile ───────────────────────────────────────────────

// StudentHealthProfile represents a student's health profile.
type StudentHealthProfile struct {
	ID                    string    `json:"id"`
	TenantID              string    `json:"-"`
	StudentID             string    `json:"student_id"`
	BloodGroup            *string   `json:"blood_group,omitempty"`
	Allergies             []string  `json:"allergies,omitempty"`
	ChronicConditions     []string  `json:"chronic_conditions,omitempty"`
	EmergencyInstructions *string   `json:"emergency_instructions,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// UpsertHealthProfilePayload is the request body for creating/updating a profile.
type UpsertHealthProfilePayload struct {
	BloodGroup            *string  `json:"blood_group,omitempty"`
	Allergies             []string `json:"allergies,omitempty"`
	ChronicConditions     []string `json:"chronic_conditions,omitempty"`
	EmergencyInstructions *string  `json:"emergency_instructions,omitempty"`
}

// ── Response Types ───────────────────────────────────────────────────────

// MedicalIncidentListResponse wraps a paginated list of medical incidents.
type MedicalIncidentListResponse struct {
	Items []MedicalIncident `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

// HealthProfileResponse wraps a health profile response.
type HealthProfileResponse struct {
	Data *StudentHealthProfile `json:"data"`
}

// StudentHealthResponse combines health profile and recent incidents.
type StudentHealthResponse struct {
	Profile   *StudentHealthProfile `json:"profile"`
	Incidents []MedicalIncident     `json:"incidents"`
}

// ── Repository Interface ─────────────────────────────────────────────────

// Repository defines the contract for health persistence.
type Repository interface {
	// Medical Incidents
	CreateIncident(ctx context.Context, params CreateIncidentParams) (string, error)
	GetIncidentByID(ctx context.Context, id, tenantID string) (*MedicalIncident, error)
	ListIncidentsByStudent(ctx context.Context, studentID, tenantID string) ([]MedicalIncident, error)
	ListIncidentsBySchool(ctx context.Context, tenantID, schoolID string, limit, offset int) ([]MedicalIncident, int, error)
	UpdateIncident(ctx context.Context, id, tenantID string, payload UpdateMedicalIncidentPayload) error
	DeleteIncident(ctx context.Context, id, tenantID string) error

	// Health Profiles
	UpsertProfile(ctx context.Context, params UpsertProfileParams) (*StudentHealthProfile, error)
	GetProfileByStudent(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error)
	GetProfileByID(ctx context.Context, id, tenantID string) (*StudentHealthProfile, error)
}

// CreateIncidentParams holds validated params for creating an incident.
type CreateIncidentParams struct {
	TenantID          string
	StudentID         string
	IncidentTimestamp time.Time
	Symptoms          string
	ActionTaken       string
	LoggedBy          string
}

// UpsertProfileParams holds validated params for upserting a health profile.
type UpsertProfileParams struct {
	TenantID              string
	StudentID             string
	BloodGroup            *string
	Allergies             []string
	ChronicConditions     []string
	EmergencyInstructions *string
}
