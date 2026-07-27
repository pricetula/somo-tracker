package academicyears

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// DateOnly — a calendar date (YYYY-MM-DD) with no time component.
// Handles both direct SQL scanning of DATE columns and JSON serialization.
// ============================================================================

// DateOnly wraps time.Time to represent a date-only value.
// It implements sql.Scanner, driver.Valuer, json.Marshaler, and json.Unmarshaler
// so it works correctly through both direct pgx scanning and JSON aggregation.
type DateOnly struct{ time.Time }

// Scan implements sql.Scanner for reading DATE values from PostgreSQL.
func (d *DateOnly) Scan(src any) error {
	if src == nil {
		d.Time = time.Time{}
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		d.Time = v
		return nil
	case string:
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return fmt.Errorf("DateOnly.Scan: parse %q: %w", v, err)
		}
		d.Time = t
		return nil
	case []byte:
		t, err := time.Parse("2006-01-02", string(v))
		if err != nil {
			return fmt.Errorf("DateOnly.Scan: parse %q: %w", string(v), err)
		}
		d.Time = t
		return nil
	default:
		return fmt.Errorf("DateOnly.Scan: unsupported type %T", src)
	}
}

// Value implements driver.Valuer for writing DateOnly to PostgreSQL as a DATE.
func (d DateOnly) Value() (driver.Value, error) {
	return d.Format("2006-01-02"), nil
}

// MarshalJSON implements json.Marshaler, outputting "YYYY-MM-DD".
func (d DateOnly) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format("2006-01-02") + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler, parsing "YYYY-MM-DD".
func (d *DateOnly) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("DateOnly.UnmarshalJSON: parse %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// String returns the date in YYYY-MM-DD format.
func (d DateOnly) String() string {
	return d.Format("2006-01-02")
}

// Before reports whether d is before other.
func (d DateOnly) Before(other DateOnly) bool {
	return d.Time.Before(other.Time)
}

// After reports whether d is after other.
func (d DateOnly) After(other DateOnly) bool {
	return d.Time.After(other.Time)
}

// Equal reports whether d and other represent the same date.
func (d DateOnly) Equal(other DateOnly) bool {
	return d.Time.Equal(other.Time)
}

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
	ID        string    `db:"id"         json:"id"`
	TenantID  string    `db:"tenant_id"  json:"-"`
	SchoolID  string    `db:"school_id"  json:"-"`
	Name      string    `db:"name"       json:"name"`
	StartDate DateOnly  `db:"start_date" json:"start_date"`
	EndDate   DateOnly  `db:"end_date"   json:"end_date"`
	IsCurrent bool      `db:"is_current" json:"is_current"`
	Version   int       `db:"version"    json:"version"`
	CreatedBy string    `db:"created_by" json:"-"`
	UpdatedBy string    `db:"updated_by" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CurrentAcademicYearWithCurrentTerm struct {
	AcademicYearID        string   `json:"academic_year_id"`
	AcademicYearName      string   `json:"academic_year_name"`
	AcademicYearStartDate DateOnly `json:"academic_year_start_date"`
	AcademicYearEndDate   DateOnly `json:"academic_year_end_date"`
	AcademicTermID        string   `json:"academic_term_id"`
	AcademicTermName      string   `json:"academic_term_name"`
	AcademicTermNumber    string   `json:"academic_term_number"`
	AcademicTermStartDate DateOnly `json:"academic_term_start_date"`
	AcademicTermEndDate   DateOnly `json:"academic_term_end_date"`
	AcademicTermIsFinal   bool     `json:"academic_term_is_final"`
}

// AcademicYearWithTerms extends AcademicYear with nested terms.
type AcademicYearWithTerms struct {
	AcademicYear
	Terms []AcademicTerm `json:"terms"`
}

// AcademicTerm represents a single term within an academic year.
type AcademicTerm struct {
	ID             string    `db:"id"                json:"id"`
	TenantID       string    `db:"tenant_id"         json:"-"`
	SchoolID       string    `db:"school_id"         json:"-"`
	AcademicYearID string    `db:"academic_year_id"  json:"academic_year_id"`
	Name           string    `db:"name"              json:"name"`
	TermNumber     int       `db:"term_number"       json:"term_number"`
	StartDate      DateOnly  `db:"start_date"        json:"start_date"`
	EndDate        DateOnly  `db:"end_date"          json:"end_date"`
	IsCurrent      bool      `db:"is_current"        json:"is_current"`
	IsFinal        bool      `db:"is_final"          json:"is_final"`
	Version        int       `db:"version"           json:"version"`
	CreatedBy      string    `db:"created_by"        json:"-"`
	UpdatedBy      string    `db:"updated_by"        json:"-"`
	CreatedAt      time.Time `db:"created_at"        json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"        json:"updated_at"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

// CreateYearBody is the request body for POST /api/v1/academic-years.
type CreateYearBody struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"` // "YYYY-MM-DD"
	EndDate   string `json:"end_date"`   // "YYYY-MM-DD"
}

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

// HasDependentsError is returned when a delete would orphan FK records.
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
	DeleteYear(ctx context.Context, id string) error
	ClearCurrentYear(ctx context.Context, schoolID, tenantID, excludeID, actorID string) error
	SetCurrentYear(ctx context.Context, id, tenantID, schoolID, actorID string) (bool, error)

	// Terms
	GetCurrent(ctx context.Context, tenantID, schoolID string) (CurrentAcademicYearWithCurrentTerm, error)
	ListTerms(ctx context.Context, tenantID, schoolID string, academicYearID *string) ([]AcademicTerm, error)
	GetTermByIDForUpdate(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error)
	CreateTerm(ctx context.Context, term *AcademicTerm) (string, error)
	UpdateTerm(ctx context.Context, term *AcademicTerm) error
	DeleteTerm(ctx context.Context, id string) error

	// Term strandedness check
	FindStrandedTerms(ctx context.Context, yearID string, newStart, newEnd time.Time) ([]ConflictingTerm, error)

	// Overlap check
	FindOverlappingTerms(ctx context.Context, yearID, excludeID string, startDate, endDate time.Time) ([]AcademicTerm, error)

	// Dependents check
	HasDependents(ctx context.Context, academicYearID string) (bool, error)
	HasTermDependents(ctx context.Context, termID string) (bool, error)

	// Sync current term
	SyncCurrentTerm(ctx context.Context, academicYearID string, now time.Time) error

	// GetCurrentAcademicYearID returns the ID of the current academic year for the school.
	// Returns empty string if none is set.
	GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error)

	// GetCurrentAcademicTermID returns the ID of the current active term for the given academic year.
	// Returns empty string if none is set.
	GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error)

	// Transaction helpers for composing operations
	Begin(ctx context.Context) (Tx, error)
}

// Tx wraps a database transaction for composable operations.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
