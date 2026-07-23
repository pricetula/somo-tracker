package cbcclasses

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound           = fmt.Errorf("cbcclasses not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists      = fmt.Errorf("cbcclasses already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput       = fmt.Errorf("invalid cbcclasses input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized       = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden          = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict           = fmt.Errorf("cbcclasses conflict: %w", middleware.ErrConflict)
	ErrEnrollmentConflict = fmt.Errorf("enrollment conflict: some students are already enrolled elsewhere: %w", middleware.ErrConflict)
	ErrStudentNotInClass  = fmt.Errorf("student is not enrolled in this class: %w", middleware.ErrNotFound)
)

// Repository defines the contract for class persistence.
type Repository interface {
	List(ctx context.Context, filter ClassListFilter) (*ClassListResult, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*Class, error)
	Create(ctx context.Context, params CreateClassParams) (*Class, error)
	Update(ctx context.Context, params UpdateClassParams) (*Class, error)
	BulkDelete(ctx context.Context, ids []string, tenantID, schoolID string) error
	ValidateAcademicYear(ctx context.Context, id, tenantID, schoolID string) (bool, error)
	ValidateAcademicTerm(ctx context.Context, id, academicYearID string) (bool, error)
	ValidateStream(ctx context.Context, id, tenantID, schoolID string) (bool, error)

	// Enrollment
	GetRoster(ctx context.Context, classID, tenantID, schoolID, academicTermID string, limit, offset int, search string) (*RosterListResult, error)
	BatchEnrollStudents(ctx context.Context, classID, tenantID, schoolID, academicTermID string, studentIDs []string) (int, error)
	UnenrollStudent(ctx context.Context, classID, studentID, tenantID, schoolID, academicTermID string) error
	GetAvailableStudents(ctx context.Context, filter AvailableStudentsFilter) (*AvailableStudentsResponse, error)
}

// AvailableStudentsFilter holds filtering and pagination for student search.
type AvailableStudentsFilter struct {
	TenantID       string
	SchoolID       string
	ClassID        string
	AcademicYearID string
	AcademicTermID string
	Search         string
	Page           int
	Limit          int
}

// Class represents a CBC class with its stream relationship.
type Class struct {
	ID           string    `json:"id"`
	GradeLevel   string    `json:"grade_level"`
	StreamName   string    `json:"stream_name"`
	StreamColor  string    `json:"stream_color"`
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
	GradeLevels    []string
	StreamIDs      []string
	Search         string
	Page           int
	Limit          int
}

// ClassListResult holds the paginated response for class listing.
type ClassListResult struct {
	Items []Class `json:"items"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
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

// ─── Roster / Enrollment Types ────────────────────────────────────────────

// RosterListResult holds the paginated response for roster listing.
type RosterListResult struct {
	Items []RosterEntry `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

// RosterEntry represents a single student enrolled in a class.
type RosterEntry struct {
	ID              string `json:"id"`
	FullName        string `json:"full_name"`
	AdmissionNumber string `json:"admission_number,omitempty"`
	UPINumber       string `json:"upi_number,omitempty"`
	Gender          string `json:"gender"`
	EnrolledAt      string `json:"enrolled_at,omitempty"`
}

// BatchEnrollPayload is the request body for POST /api/v1/classes/:id/enroll.
type BatchEnrollPayload struct {
	StudentIDs     []string `json:"student_ids"`
	AcademicTermID string   `json:"academic_term_id"`
}

// BatchEnrollResponse is returned after a successful batch enrollment.
type BatchEnrollResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	EnrolledCount int    `json:"enrolled_count"`
}

// AvailableStudent represents a student who can be enrolled in the class.
type AvailableStudent struct {
	ID              string  `json:"id"`
	FullName        string  `json:"full_name"`
	AdmissionNumber *string `json:"admission_number,omitempty"`
	UPINumber       *string `json:"upi_number,omitempty"`
	Gender          string  `json:"gender"`
	CurrentClass    *string `json:"current_class,omitempty"`
	CurrentClassID  *string `json:"current_class_id,omitempty"`
}

// AvailableStudentsResponse holds the paginated list of available students.
type AvailableStudentsResponse struct {
	Items []AvailableStudent `json:"items"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}
