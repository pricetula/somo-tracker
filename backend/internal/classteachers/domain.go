// Package classteachers manages teacher assignments to classes and learning areas.
// Supports PRIMARY_CLASS_TEACHER (one per class) and SUBJECT_TEACHER (one per
// class + learning_area combination). Validates constraints at the service layer.
package classteachers

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors.
var (
	ErrNotFound               = xerrors.NotFound("class teacher not found")
	ErrAlreadyExists          = xerrors.AlreadyExists("class teacher already exists")
	ErrInvalidInput           = xerrors.InvalidInput("invalid class teacher input")
	ErrUnauthorized           = xerrors.Unauthorized("unauthorized")
	ErrForbidden              = xerrors.Forbidden("forbidden")
	ErrConflict               = xerrors.Conflict("class teacher conflict")
	ErrPrimaryAlreadyAssigned = xerrors.Conflict("class already has a primary teacher")
)

// ── Domain Models ────────────────────────────────────────────────────────

// ClassTeacher represents a teacher assigned to a class.
type ClassTeacher struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"-"`
	ClassID        string    `json:"class_id"`
	UserID         string    `json:"user_id"`
	TeacherName    string    `json:"teacher_name,omitempty"`
	LearningAreaID *string   `json:"learning_area_id,omitempty"`
	LearningArea   *string   `json:"learning_area,omitempty"`
	TeacherRole    string    `json:"teacher_role"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateClassTeacherPayload is the request body for POST.
type CreateClassTeacherPayload struct {
	UserID         string  `json:"user_id"`
	ClassID        string  `json:"class_id"`
	LearningAreaID *string `json:"learning_area_id,omitempty"`
	TeacherRole    string  `json:"teacher_role"`
}

// ClassTeacherListResponse wraps a list of class teacher assignments.
type ClassTeacherListResponse struct {
	Items []ClassTeacher `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// ── Repository Interface ─────────────────────────────────────────────────

// Repository defines the contract for class teacher persistence.
type Repository interface {
	Create(ctx context.Context, params CreateClassTeacherParams) (string, error)
	GetByID(ctx context.Context, id, tenantID string) (*ClassTeacher, error)
	ListByClass(ctx context.Context, classID, tenantID string) ([]ClassTeacher, error)
	ListByTeacher(ctx context.Context, userID, tenantID string) ([]ClassTeacher, error)
	Delete(ctx context.Context, id, tenantID string) error
	CountPrimaryForClass(ctx context.Context, classID, tenantID string) (int, error)
	ExistsForSubject(ctx context.Context, classID, userID, learningAreaID, tenantID string) (bool, error)
}

// CreateClassTeacherParams holds validated parameters for creating an assignment.
type CreateClassTeacherParams struct {
	TenantID       string
	ClassID        string
	UserID         string
	LearningAreaID *string
	TeacherRole    string
}
