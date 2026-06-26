package cbcclasses

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound            = fmt.Errorf("cbcclasses not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists       = fmt.Errorf("cbcclasses already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput        = fmt.Errorf("invalid cbcclasses input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized        = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden           = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict            = fmt.Errorf("cbcclasses conflict: %w", middleware.ErrConflict)
	ErrClassLocked         = fmt.Errorf("cbcclasses locked: %w", middleware.ErrConflict)
	ErrClassHasAssessments = fmt.Errorf("cbcclasses has assessments: %w", middleware.ErrConflict)
)

// Repository defines the contract for class persistence.
type Repository interface {
	List(ctx context.Context, filter ClassListFilter) (*ClassListResult, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*Class, error)
	Create(ctx context.Context, params CreateClassParams) (*Class, error)
	Update(ctx context.Context, params UpdateClassParams) (*Class, error)
	BulkDelete(ctx context.Context, ids []string, tenantID, schoolID string) error
	HasAssessmentSessions(ctx context.Context, classID, tenantID string) (bool, error)
	HasAnyAssessmentSessions(ctx context.Context, classIDs []string, tenantID string) (bool, error)
	ValidateAcademicYear(ctx context.Context, id, tenantID, schoolID string) (bool, error)
	ValidateAcademicTerm(ctx context.Context, id, academicYearID string) (bool, error)
	ValidateStream(ctx context.Context, id, tenantID, schoolID string) (bool, error)
}

// Class represents a CBC class with its stream relationship.
type Class struct {
	ID           string    `json:"id"`
	GradeLevel   string    `json:"grade_level"`
	StreamName   string    `json:"stream_name"`
	DisplayLabel string    `json:"display_label"`
	StreamID     string    `json:"stream_id"`
	StudentCount int       `json:"student_count,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// ClassListFilter holds filtering and pagination params for listing classes.
type ClassListFilter struct {
	TenantID       string
	SchoolID       string
	AcademicYearID string
	AcademicTermID string
	GradeLevel     *string
	StreamID       *string
	Page           int
	Limit          int
}

// ClassListResult holds the paginated response for class listing.
type ClassListResult struct {
	Data         []Class `json:"data"`
	TotalRecords int     `json:"total_records"`
	CurrentPage  int     `json:"current_page"`
	Limit        int     `json:"limit"`
	TotalPages   int     `json:"total_pages"`
}

// CreateClassPayload is the request body for POST /api/v1/classes.
type CreateClassPayload struct {
	GradeLevel     string   `json:"grade_level"`
	AcademicYearID string   `json:"academic_year_id"`
	AcademicTermID string   `json:"academic_term_id"`
	StreamID       string   `json:"stream_id"`
	StudentIDs     []string `json:"student_ids"`
}

// CreateClassParams holds validated params for creating a class.
type CreateClassParams struct {
	TenantID       string
	SchoolID       string
	AcademicYearID string
	AcademicTermID string
	GradeLevel     string
	StreamID       string
	StudentIDs     []string
}

// UpdateClassPayload is the request body for PUT /api/v1/classes/:id.
type UpdateClassPayload struct {
	GradeLevel     string   `json:"grade_level"`
	StreamID       string   `json:"stream_id"`
	AcademicTermID string   `json:"academic_term_id"`
	StudentIDs     []string `json:"student_ids"`
}

// UpdateClassParams holds validated params for updating a class.
type UpdateClassParams struct {
	ClassID        string
	TenantID       string
	SchoolID       string
	GradeLevel     string
	StreamID       string
	AcademicTermID string
	StudentIDs     []string
}

// BulkDeletePayload is the request body for DELETE /api/v1/classes.
type BulkDeletePayload struct {
	ClassIDs []string `json:"class_ids"`
}
