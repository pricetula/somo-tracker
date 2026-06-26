package academicyears

import (
	"context"
	"errors"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound      = fmt.Errorf("academicyears not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("academicyears already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid academicyears input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("academicyears conflict: %w", middleware.ErrConflict)
)

// Module-specific sentinels.
var (
	ErrTermsOutOfRange     = errors.New("terms_out_of_range")
	ErrHasDependents       = errors.New("has_dependents")
	ErrTermOutOfYearBounds = errors.New("term_out_of_year_bounds")
	ErrTermDateOverlap     = errors.New("term_date_overlap")
	ErrTermNumberExists    = errors.New("term_number_exists")
)

// ============================================================================
// Domain Models
// ============================================================================

// AcademicYear represents a single academic year in a school.
type AcademicYear struct {
	ID        string     `db:"id"         json:"id"`
	TenantID  string     `db:"tenant_id"  json:"-"`
	SchoolID  string     `db:"school_id"  json:"-"`
	Name      string     `db:"name"       json:"name"`
	StartDate time.Time  `db:"start_date" json:"start_date"`
	EndDate   time.Time  `db:"end_date"   json:"end_date"`
	IsCurrent bool       `db:"is_current" json:"is_current"`
	Version   int        `db:"version"    json:"version"`
	CreatedBy string     `db:"created_by" json:"-"`
	UpdatedBy string     `db:"updated_by" json:"-"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

// AcademicYearWithTerms extends AcademicYear with nested terms.
type AcademicYearWithTerms struct {
	AcademicYear
	Terms []AcademicTerm `json:"terms"`
}

// AcademicTerm represents a single term within an academic year.
type AcademicTerm struct {
	ID             string     `db:"id"                json:"id"`
	TenantID       string     `db:"tenant_id"         json:"-"`
	SchoolID       string     `db:"school_id"         json:"-"`
	AcademicYearID string     `db:"academic_year_id"  json:"academic_year_id"`
	Name           string     `db:"name"              json:"name"`
	TermNumber     int        `db:"term_number"       json:"term_number"`
	StartDate      time.Time  `db:"start_date"        json:"start_date"`
	EndDate        time.Time  `db:"end_date"          json:"end_date"`
	IsCurrent      bool       `db:"is_current"        json:"is_current"`
	IsFinal        bool       `db:"is_final"          json:"is_final"`
	Version        int        `db:"version"           json:"version"`
	CreatedBy      string     `db:"created_by"        json:"-"`
	UpdatedBy      string     `db:"updated_by"        json:"-"`
	CreatedAt      time.Time  `db:"created_at"        json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"        json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at"        json:"-"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

// PatchYearBody is the allowed request body for PATCH /api/v1/academic-years/:id.
type PatchYearBody struct {
	Name      *string `json:"name,omitempty"`
	StartDate *string `json:"start_date,omitempty"` // "YYYY-MM-DD"
	EndDate   *string `json:"end_date,omitempty"`
	Version   *int    `json:"version"` // required for optimistic lock
}

// SetCurrentResponse is the body for POST .../:id/set-current.
type SetCurrentResponse struct {
	Message string `json:"message"`
}

// CreateTermBody is the request body for POST /api/v1/academic-terms.
type CreateTermBody struct {
	AcademicYearID string `json:"academic_year_id"`
	Name           string `json:"name"`
	TermNumber     int    `json:"term_number"`
	StartDate      string `json:"start_date"` // "YYYY-MM-DD"
	EndDate        string `json:"end_date"`
}

// PatchTermBody is the allowed request body for PATCH /api/v1/academic-terms/:id.
type PatchTermBody struct {
	Name      *string `json:"name,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
	Version   *int    `json:"version"` // required for optimistic lock
}

// ConflictingTerm is returned in 422 responses when dates strand terms.
type ConflictingTerm struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// ============================================================================
// Error Wrappers for Rich HTTP Responses
// ============================================================================

// TermsOutOfRangeError carries conflicting term data for 422 responses.
type TermsOutOfRangeError struct {
	ConflictingTerms []ConflictingTerm
}

func (e *TermsOutOfRangeError) Error() string {
	return "the new date range would strand existing terms"
}

// HasDependentsError is returned when a soft-delete would orphan FK records.
type HasDependentsError struct {
	Message string
}

func (e *HasDependentsError) Error() string {
	return e.Message
}

// TermOutOfYearBoundsError is returned when a term falls outside its parent year.
type TermOutOfYearBoundsError struct{}

func (e *TermOutOfYearBoundsError) Error() string {
	return "term dates must be within the academic year"
}

// TermDateOverlapError names the conflicting term.
type TermDateOverlapError struct {
	ConflictingName string
}

func (e *TermDateOverlapError) Error() string {
	return fmt.Sprintf("term dates overlap with %q", e.ConflictingName)
}

// TermNumberExistsError is returned on duplicate term_number.
type TermNumberExistsError struct{}

func (e *TermNumberExistsError) Error() string {
	return "a term with this number already exists in this academic year"
}

// ============================================================================
// Repository Interface
// ============================================================================

// Repository defines the contract for academic year and term persistence.
type Repository interface {
	// Years
	ListYears(ctx context.Context, tenantID, schoolID string) ([]AcademicYearWithTerms, error)
	GetYearByID(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error)
	GetYearByIDForUpdate(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error)
	CreateYear(ctx context.Context, year *AcademicYear) (string, error)
	UpdateYear(ctx context.Context, year *AcademicYear) error
	SoftDeleteYear(ctx context.Context, id, actorID string) error
	ClearCurrentYear(ctx context.Context, schoolID, tenantID, excludeID, actorID string) error
	SetCurrentYear(ctx context.Context, id, tenantID, schoolID, actorID string) (bool, error)

	// Terms
	ListTerms(ctx context.Context, tenantID, schoolID string, academicYearID *string) ([]AcademicTerm, error)
	GetTermByIDForUpdate(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error)
	CreateTerm(ctx context.Context, term *AcademicTerm) (string, error)
	UpdateTerm(ctx context.Context, term *AcademicTerm) error
	SoftDeleteTerm(ctx context.Context, id, actorID string) error

	// Term strandedness check
	FindStrandedTerms(ctx context.Context, yearID string, newStart, newEnd time.Time) ([]ConflictingTerm, error)

	// Overlap check
	FindOverlappingTerms(ctx context.Context, yearID, excludeID string, startDate, endDate time.Time) ([]AcademicTerm, error)

	// Dependents check
	HasDependents(ctx context.Context, academicYearID string) (bool, error)
	HasTermDependents(ctx context.Context, termID string) (bool, error)

	// Sync current term
	SyncCurrentTerm(ctx context.Context, academicYearID string, now time.Time) error

	// Transaction helpers for composing operations
	Begin(ctx context.Context) (Tx, error)
}

// Tx wraps a database transaction for composable operations.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
