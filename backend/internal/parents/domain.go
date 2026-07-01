package parents

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound        = fmt.Errorf("parent not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists   = fmt.Errorf("parent already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput    = fmt.Errorf("invalid parent input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized    = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden       = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict        = fmt.Errorf("parent conflict: %w", middleware.ErrConflict)
	ErrDuplicateLink   = fmt.Errorf("student already linked to this parent: %w", middleware.ErrConflict)
	ErrStudentNotFound = fmt.Errorf("student not found: %w", middleware.ErrNotFound)
)

// ============================================================================
// Domain Models
// ============================================================================

// Parent represents a parent/guardian profile linked to a platform user.
type Parent struct {
	ID          string `json:"id"`
	TenantID    string `json:"-"`
	UserID      string `json:"user_id"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
}

// ParentDetail extends Parent with linked students.
type ParentDetail struct {
	Parent
	LinkedStudents []StudentLink `json:"linked_students"`
}

// StudentLink represents a linked student in a parent detail response.
type StudentLink struct {
	StudentID    string  `json:"student_id"`
	FullName     string  `json:"full_name"`
	Relationship *string `json:"relationship,omitempty"`
	IsPrimary    bool    `json:"is_primary"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

type CreateParentPayload struct {
	Email       string `json:"email"`
	FullName    string `json:"full_name"`
	PhoneNumber string `json:"phone_number"`
}

type UpdateParentPayload struct {
	PhoneNumber *string `json:"phone_number,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type LinkStudentPayload struct {
	StudentID    string  `json:"student_id"`
	Relationship *string `json:"relationship,omitempty"`
	IsPrimary    *bool   `json:"is_primary,omitempty"` // default: false
}

type CreateParentResponse struct {
	ID string `json:"id"`
}

type ParentDetailResponse struct {
	Data ParentDetail `json:"data"`
}

type ListParentsResponse struct {
	Data []Parent `json:"data"`
}

// ============================================================================
// Cross-Domain Interface: StudentResolver
// ============================================================================

// StudentResolver validates student existence and tenant membership.
// The parents PgRepository implements this interface.
type StudentResolver interface {
	StudentExistsInTenant(ctx context.Context, studentID, tenantID string) (bool, error)
}

// ============================================================================
// Repository Interface
// ============================================================================

// Repository defines the contract for parent persistence.
type Repository interface {
	// Create parent profile (creates user if needed)
	Create(ctx context.Context, tenantID string, payload CreateParentPayload) (string, error)

	// GetByID retrieves a parent by primary key.
	GetByID(ctx context.Context, id, tenantID string) (*Parent, error)

	// GetDetail retrieves a parent with linked students.
	GetDetail(ctx context.Context, id, tenantID string) (*ParentDetail, error)

	// List returns parents filtered by search or student_id.
	List(ctx context.Context, tenantID string, search, studentID string) ([]Parent, error)

	// Update applies partial updates to a parent profile.
	Update(ctx context.Context, id, tenantID string, payload UpdateParentPayload) error

	// Delete removes a parent profile (user record preserved).
	Delete(ctx context.Context, id, tenantID string) error

	// LinkStudent links a student to a parent.
	LinkStudent(ctx context.Context, parentID, tenantID string, payload LinkStudentPayload) error

	// UnlinkStudent removes a student-parent link.
	UnlinkStudent(ctx context.Context, parentID, studentID, tenantID string) error

	// DemotePrimaryForStudent clears the is_primary flag for all parents
	// linked to the given student within the tenant.
	DemotePrimaryForStudent(ctx context.Context, studentID, tenantID string) error

	// CountLinksByStudent returns the number of parents linked to a student.
	CountLinksByStudent(ctx context.Context, studentID, tenantID string) (int, error)
}
