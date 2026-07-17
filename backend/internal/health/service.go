package health

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service contains business logic for health records.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ═══════════════════════════════════════════════════════════════════════════
// MEDICAL INCIDENTS
// ═══════════════════════════════════════════════════════════════════════════

// CreateIncident logs a new medical incident.
func (s *Service) CreateIncident(ctx context.Context, tenantID, loggedBy string, payload CreateMedicalIncidentPayload) (*MedicalIncident, error) {
	if tenantID == "" || loggedBy == "" {
		return nil, fmt.Errorf("health.Service.CreateIncident: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(payload.StudentID) == "" {
		return nil, fmt.Errorf("health.Service.CreateIncident: student_id is required: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(payload.Symptoms) == "" {
		return nil, fmt.Errorf("health.Service.CreateIncident: symptoms is required: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(payload.ActionTaken) == "" {
		return nil, fmt.Errorf("health.Service.CreateIncident: action_taken is required: %w", ErrInvalidInput)
	}

	ts := time.Now()
	if payload.IncidentTimestamp != "" {
		parsed, err := time.Parse(time.RFC3339, payload.IncidentTimestamp)
		if err != nil {
			return nil, fmt.Errorf("health.Service.CreateIncident: invalid incident_timestamp: %w", ErrInvalidInput)
		}
		ts = parsed
	}

	params := CreateIncidentParams{
		TenantID:          tenantID,
		StudentID:         strings.TrimSpace(payload.StudentID),
		IncidentTimestamp: ts,
		Symptoms:          strings.TrimSpace(payload.Symptoms),
		ActionTaken:       strings.TrimSpace(payload.ActionTaken),
		LoggedBy:          loggedBy,
	}

	id, err := s.repo.CreateIncident(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("health.Service.CreateIncident: %w", err)
	}

	incident, err := s.repo.GetIncidentByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("health.Service.CreateIncident: %w", err)
	}
	return incident, nil
}

// GetIncidentByID retrieves a medical incident.
func (s *Service) GetIncidentByID(ctx context.Context, id, tenantID string) (*MedicalIncident, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("health.Service.GetIncidentByID: %w", ErrInvalidInput)
	}
	return s.repo.GetIncidentByID(ctx, id, tenantID)
}

// ListIncidentsByStudent returns all incidents for a student.
func (s *Service) ListIncidentsByStudent(ctx context.Context, studentID, tenantID string) (*MedicalIncidentListResponse, error) {
	if studentID == "" || tenantID == "" {
		return nil, fmt.Errorf("health.Service.ListIncidentsByStudent: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListIncidentsByStudent(ctx, studentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("health.Service.ListIncidentsByStudent: %w", err)
	}
	return &MedicalIncidentListResponse{
		Items: items,
		Total: len(items),
		Page:  1,
		Limit: len(items),
	}, nil
}

// ListIncidentsBySchool returns paginated incidents for a school.
func (s *Service) ListIncidentsBySchool(ctx context.Context, tenantID, schoolID string, page, limit int) (*MedicalIncidentListResponse, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("health.Service.ListIncidentsBySchool: %w", ErrInvalidInput)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	items, total, err := s.repo.ListIncidentsBySchool(ctx, tenantID, schoolID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("health.Service.ListIncidentsBySchool: %w", err)
	}
	return &MedicalIncidentListResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// UpdateIncident updates a medical incident.
func (s *Service) UpdateIncident(ctx context.Context, id, tenantID string, payload UpdateMedicalIncidentPayload) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("health.Service.UpdateIncident: %w", ErrInvalidInput)
	}
	if payload.Symptoms == nil && payload.ActionTaken == nil {
		return fmt.Errorf("health.Service.UpdateIncident: at least one field to update is required: %w", ErrInvalidInput)
	}
	return s.repo.UpdateIncident(ctx, id, tenantID, payload)
}

// DeleteIncident deletes a medical incident.
func (s *Service) DeleteIncident(ctx context.Context, id, tenantID string) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("health.Service.DeleteIncident: %w", ErrInvalidInput)
	}
	return s.repo.DeleteIncident(ctx, id, tenantID)
}

// ═══════════════════════════════════════════════════════════════════════════
// STUDENT HEALTH PROFILES
// ═══════════════════════════════════════════════════════════════════════════

// UpsertProfile creates or updates a student health profile.
func (s *Service) UpsertProfile(ctx context.Context, tenantID string, studentID string, payload UpsertHealthProfilePayload) (*StudentHealthProfile, error) {
	if tenantID == "" || studentID == "" {
		return nil, fmt.Errorf("health.Service.UpsertProfile: %w", ErrInvalidInput)
	}

	var bg *string
	if payload.BloodGroup != nil {
		t := strings.TrimSpace(*payload.BloodGroup)
		if t != "" {
			bg = &t
		}
	}

	allergies := payload.Allergies
	if allergies == nil {
		allergies = []string{}
	}
	chronic := payload.ChronicConditions
	if chronic == nil {
		chronic = []string{}
	}

	var ei *string
	if payload.EmergencyInstructions != nil {
		t := strings.TrimSpace(*payload.EmergencyInstructions)
		if t != "" {
			ei = &t
		}
	}

	params := UpsertProfileParams{
		TenantID:              tenantID,
		StudentID:             studentID,
		BloodGroup:            bg,
		Allergies:             allergies,
		ChronicConditions:     chronic,
		EmergencyInstructions: ei,
	}

	profile, err := s.repo.UpsertProfile(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("health.Service.UpsertProfile: %w", err)
	}
	return profile, nil
}

// GetProfileByStudent retrieves a health profile for a student.
func (s *Service) GetProfileByStudent(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error) {
	if studentID == "" || tenantID == "" {
		return nil, fmt.Errorf("health.Service.GetProfileByStudent: %w", ErrInvalidInput)
	}
	return s.repo.GetProfileByStudent(ctx, studentID, tenantID)
}

// GetStudentHealth combines profile and recent incidents into one response.
func (s *Service) GetStudentHealth(ctx context.Context, studentID, tenantID string) (*StudentHealthResponse, error) {
	if studentID == "" || tenantID == "" {
		return nil, fmt.Errorf("health.Service.GetStudentHealth: %w", ErrInvalidInput)
	}

	profile, _ := s.repo.GetProfileByStudent(ctx, studentID, tenantID)

	incidents, err := s.repo.ListIncidentsByStudent(ctx, studentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("health.Service.GetStudentHealth: %w", err)
	}

	return &StudentHealthResponse{
		Profile:   profile,
		Incidents: incidents,
	}, nil
}
