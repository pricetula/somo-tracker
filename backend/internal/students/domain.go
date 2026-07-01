package students

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound         = fmt.Errorf("student not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists    = fmt.Errorf("student already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput     = fmt.Errorf("invalid student input: %w", middleware.ErrInvalidInput)
	ErrDuplicateUPI     = fmt.Errorf("duplicate UPI number: %w", middleware.ErrAlreadyExists)
	ErrDuplicateEnroll  = fmt.Errorf("student already enrolled in this term: %w", middleware.ErrConflict)
	ErrStudentNotActive = fmt.Errorf("student is not active: %w", middleware.ErrInvalidInput)
	ErrForbidden        = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
)

// ============================================================================
// Domain Models
// ============================================================================

// Student represents a full student record.
type Student struct {
	ID                   string  `json:"id"`
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	ClassName            *string `json:"class_name,omitempty"`
	ClassID              *string `json:"class_id,omitempty"`
	IsActive             bool    `json:"is_active"`
	CreatedAt            string  `json:"created_at"`
}

// StudentDetail extends Student with enrollment history.
type StudentDetail struct {
	Student
	Enrollments []Enrollment `json:"enrollments"`
}

// Enrollment represents a single term enrollment record.
type Enrollment struct {
	ID             string `json:"id"`
	StudentID      string `json:"student_id"`
	ClassID        string `json:"class_id"`
	AcademicTermID string `json:"academic_term_id"`
	TermName       string `json:"term_name"`
	TermNumber     int    `json:"term_number"`
	AcademicYear   string `json:"academic_year"`
	ClassName      string `json:"class_name"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

type ListStudentsResponse struct {
	Students []Student `json:"students"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}

type StudentDetailResponse struct {
	Data StudentDetail `json:"data"`
}

type CreateStudentPayload struct {
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	ClassID              *string `json:"class_id,omitempty"`
}

type UpdateStudentPayload struct {
	FullName             *string `json:"full_name,omitempty"`
	Gender               *string `json:"gender,omitempty"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	IsActive             *bool   `json:"is_active,omitempty"`
}

type CreateStudentResponse struct {
	ID string `json:"id"`
}

type CreateEnrollmentPayload struct {
	AcademicTermID string `json:"academic_term_id"`
	ClassID        string `json:"class_id"`
	Status         string `json:"status,omitempty"` // defaults to ACTIVE
}

type CreateEnrollmentResponse struct {
	ID string `json:"id"`
}

type ListEnrollmentsResponse struct {
	Data []Enrollment `json:"data"`
}

// ListFilter holds query parameters for listing students.
type ListFilter struct {
	TenantID string
	SchoolID string
	Page     int
	Limit    int
	Search   string
	ClassID  string
	Gender   string
}

// ============================================================================
// Repository Interface
// ============================================================================

// StudentRepository defines the data access contract.
type StudentRepository interface {
	List(ctx context.Context, filter ListFilter) ([]Student, int, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*Student, error)
	Create(ctx context.Context, student *Student) (string, error)
	Update(ctx context.Context, student *Student) error
	GetDetail(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error)

	// Enrollments
	CreateEnrollment(ctx context.Context, enrollment *Enrollment) (string, error)
	ListEnrollments(ctx context.Context, studentID, tenantID string) ([]Enrollment, error)
	IsEnrolledInTerm(ctx context.Context, studentID, academicTermID, tenantID string) (bool, error)
}
