package parents

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
)

// Service contains business logic for parents.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// validateEmail checks if the provided string is a valid email address.
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// validatePhoneNumber performs basic phone number validation.
// Phone number must be non-empty and contain only digits, +, -, and spaces.
func validatePhoneNumber(phone string) bool {
	if phone == "" {
		return false
	}
	for _, c := range phone {
		if (c < '0' || c > '9') && c != '+' && c != '-' && c != ' ' {
			return false
		}
	}
	return true
}

// ============================================================================
// CREATE
// ============================================================================

// Create creates a new parent profile linked to a platform user.
func (s *Service) Create(ctx context.Context, tenantID string, payload CreateParentPayload) (*Parent, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("parents.Service.Create: %w", ErrInvalidInput)
	}

	// Validate required fields
	if strings.TrimSpace(payload.Email) == "" {
		return nil, fmt.Errorf("parents.Service.Create: email is required: %w", ErrInvalidInput)
	}
	if !validateEmail(payload.Email) {
		return nil, fmt.Errorf("parents.Service.Create: invalid email %q: %w", payload.Email, ErrInvalidInput)
	}
	if strings.TrimSpace(payload.FullName) == "" {
		return nil, fmt.Errorf("parents.Service.Create: full_name is required: %w", ErrInvalidInput)
	}
	if !validatePhoneNumber(payload.PhoneNumber) {
		return nil, fmt.Errorf("parents.Service.Create: invalid or empty phone_number: %w", ErrInvalidInput)
	}

	id, err := s.Repo.Create(ctx, tenantID, payload)
	if err != nil {
		return nil, fmt.Errorf("parents.Service.Create: %w", err)
	}

	// Fetch the created parent to return full details
	parent, err := s.Repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("parents.Service.Create: %w", err)
	}

	slog.Info("parent.created",
		"tenant_id", tenantID,
		"resource_id", id,
		"email", payload.Email,
		"full_name", payload.FullName,
	)

	return parent, nil
}

// ============================================================================
// GET BY ID
// ============================================================================

// GetByID returns a single parent by ID.
func (s *Service) GetByID(ctx context.Context, id, tenantID string) (*Parent, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("parents.Service.GetByID: %w", ErrInvalidInput)
	}
	return s.Repo.GetByID(ctx, id, tenantID)
}

// ============================================================================
// GET DETAIL
// ============================================================================

// GetDetail returns a parent with linked students.
func (s *Service) GetDetail(ctx context.Context, id, tenantID string) (*ParentDetail, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("parents.Service.GetDetail: %w", ErrInvalidInput)
	}
	return s.Repo.GetDetail(ctx, id, tenantID)
}

// ============================================================================
// LIST
// ============================================================================

// List returns parents optionally filtered by search or student_id.
func (s *Service) List(ctx context.Context, tenantID string, search, studentID string) ([]Parent, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("parents.Service.List: %w", ErrInvalidInput)
	}
	return s.Repo.List(ctx, tenantID, search, studentID)
}

// ============================================================================
// UPDATE
// ============================================================================

// Update applies partial updates to a parent profile.
func (s *Service) Update(ctx context.Context, id, tenantID string, payload UpdateParentPayload) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("parents.Service.Update: %w", ErrInvalidInput)
	}

	// At least one field must be provided
	if payload.PhoneNumber == nil && payload.IsActive == nil {
		return fmt.Errorf("parents.Service.Update: at least one of phone_number or is_active must be provided: %w", ErrInvalidInput)
	}

	// Validate phone_number if set
	if payload.PhoneNumber != nil {
		if !validatePhoneNumber(*payload.PhoneNumber) {
			return fmt.Errorf("parents.Service.Update: invalid phone_number: %w", ErrInvalidInput)
		}
	}

	if err := s.Repo.Update(ctx, id, tenantID, payload); err != nil {
		return fmt.Errorf("parents.Service.Update: %w", err)
	}

	slog.Info("parent.updated",
		"tenant_id", tenantID,
		"resource_id", id,
		"phone_number_changed", payload.PhoneNumber != nil,
		"is_active_changed", payload.IsActive != nil,
	)

	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// Delete removes a parent profile. The linked user record is preserved.
func (s *Service) Delete(ctx context.Context, id, tenantID string) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("parents.Service.Delete: %w", ErrInvalidInput)
	}
	return s.Repo.Delete(ctx, id, tenantID)
}

// ============================================================================
// LINK STUDENT
// ============================================================================

// LinkStudent links a student to a parent.
func (s *Service) LinkStudent(ctx context.Context, parentID, tenantID string, payload LinkStudentPayload) error {
	if parentID == "" || tenantID == "" {
		return fmt.Errorf("parents.Service.LinkStudent: %w", ErrInvalidInput)
	}
	if payload.StudentID == "" {
		return fmt.Errorf("parents.Service.LinkStudent: student_id is required: %w", ErrInvalidInput)
	}

	// Verify parent exists
	_, err := s.Repo.GetByID(ctx, parentID, tenantID)
	if err != nil {
		return fmt.Errorf("parents.Service.LinkStudent: %w", err)
	}

	if err := s.Repo.LinkStudent(ctx, parentID, tenantID, payload); err != nil {
		return fmt.Errorf("parents.Service.LinkStudent: %w", err)
	}

	slog.Info("parent.student.linked",
		"tenant_id", tenantID,
		"parent_id", parentID,
		"student_id", payload.StudentID,
		"is_primary", payload.IsPrimary,
	)

	return nil
}

// ============================================================================
// UNLINK STUDENT
// ============================================================================

// UnlinkStudent removes a student-parent link.
func (s *Service) UnlinkStudent(ctx context.Context, parentID, studentID, tenantID string) error {
	if parentID == "" || studentID == "" || tenantID == "" {
		return fmt.Errorf("parents.Service.UnlinkStudent: %w", ErrInvalidInput)
	}
	return s.Repo.UnlinkStudent(ctx, parentID, studentID, tenantID)
}
