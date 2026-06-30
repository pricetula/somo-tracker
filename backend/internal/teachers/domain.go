package teachers

import (
	"context"
	"errors"
	"time"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("teachers not found")
	ErrAlreadyExists = errors.New("teachers already exists")
	ErrInvalidInput  = errors.New("invalid teachers input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("teachers conflict")
)

// Repository defines the contract for teacher persistence.
type Repository interface {
	ListBySchool(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error)
	ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error
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
	Teachers []Teacher `json:"teachers"`
	Total    int       `json:"total"`
}

// ToggleActiveRequest is the payload for activating/deactivating a teacher.
type ToggleActiveRequest struct {
	IsActive bool `json:"is_active"`
}
