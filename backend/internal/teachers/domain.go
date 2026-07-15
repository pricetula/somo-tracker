package teachers

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound      = fmt.Errorf("teachers not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("teachers already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid teachers input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("teachers conflict: %w", middleware.ErrConflict)
)

// Repository defines the contract for teacher persistence.
type Repository interface {
	ListBySchool(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error)
	GetByID(ctx context.Context, userID, tenantID, schoolID string) (*Teacher, error)
	Update(ctx context.Context, userID, tenantID, schoolID string, payload UpdateTeacherPayload) error
	ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error
	Delete(ctx context.Context, tenantID, schoolID, userID string) error
}

// Teacher represents a user with the TEACHER role, including
// educator-specific fields.
type Teacher struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	FullName          string    `json:"full_name"`
	TSCNumber         *string   `json:"tsc_number"`
	KNECPanelAssessor *string   `json:"knec_panel_assessor_id"`
	TeacherRole       *string   `json:"teacher_role"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
}

// ListResponse wraps a paginated teacher list.
type ListResponse struct {
	Items []Teacher `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

// ToggleActiveRequest is the payload for activating/deactivating a teacher.
type ToggleActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateTeacherPayload is the payload for updating a teacher's profile.
type UpdateTeacherPayload struct {
	FullName          *string `json:"full_name,omitempty"`
	TSCNumber         *string `json:"tsc_number,omitempty"`
	KNECPanelAssessor *string `json:"knec_panel_assessor_id,omitempty"`
}
