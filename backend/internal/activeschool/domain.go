package activeschool

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
// Each wraps the corresponding middleware sentinel so that middleware.HTTPError
// can match them via errors.Is.
var (
	ErrNotFound      = fmt.Errorf("activeschool not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("activeschool already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid activeschool input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("activeschool conflict: %w", middleware.ErrConflict)
)

// Repository defines the contract for member_active_school persistence.
type Repository interface {
	Upsert(ctx context.Context, tenantID, userID, schoolID string) error
	GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error)
}

// MemberActiveSchool represents a row in member_active_school.
type MemberActiveSchool struct {
	UserID     string    `json:"user_id"`
	TenantID   string    `json:"tenant_id"`
	SchoolID   string    `json:"school_id"`
	SwitchedAt time.Time `json:"switched_at"`
}

// SwitchActiveSchoolPayload is the request body for PUT /api/v1/active-school.
type SwitchActiveSchoolPayload struct {
	SchoolID string `json:"school_id"`
}
