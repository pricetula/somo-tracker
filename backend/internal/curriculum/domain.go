package curriculum

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound      = fmt.Errorf("curriculum not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("curriculum already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid curriculum input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("curriculum conflict: %w", middleware.ErrConflict)
)

// Repository defines the contract for learning area persistence.
type Repository interface {
	Create(ctx context.Context, params CreateLearningAreaParams) (string, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error)
	List(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error)
	Update(ctx context.Context, params UpdateLearningAreaParams) error
	Delete(ctx context.Context, id, tenantID, schoolID string) error
}

// LearningArea represents a CBC learning area (subject) taught at a school.
type LearningArea struct {
	ID             string `json:"id"`
	TenantID       string `json:"-"`
	SchoolID       string `json:"-"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	EducationLevel string `json:"education_level"`
}

// CreateLearningAreaParams holds the fields needed to create a learning area.
type CreateLearningAreaParams struct {
	TenantID       string
	SchoolID       string
	Name           string
	Code           string
	EducationLevel string
}

// UpdateLearningAreaParams holds fields that can be updated on a learning area.
type UpdateLearningAreaParams struct {
	ID             string
	TenantID       string
	SchoolID       string
	Name           *string
	Code           *string
	EducationLevel *string
}

// CreateLearningAreaPayload is the request body for POST /api/v1/curriculum/learning-areas.
type CreateLearningAreaPayload struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	EducationLevel string `json:"education_level"`
}

// UpdateLearningAreaPayload is the request body for PUT /api/v1/curriculum/learning-areas/:id.
type UpdateLearningAreaPayload struct {
	Name           *string `json:"name,omitempty"`
	Code           *string `json:"code,omitempty"`
	EducationLevel *string `json:"education_level,omitempty"`
}

// ListLearningAreasResponse wraps a list of learning areas.
type ListLearningAreasResponse struct {
	LearningAreas []LearningArea `json:"learning_areas"`
	Total         int            `json:"total"`
}
