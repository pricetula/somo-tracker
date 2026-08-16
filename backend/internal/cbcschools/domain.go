package cbcschools

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors.
var (
	ErrNotFound      = xerrors.NotFound("cbcschool not found")
	ErrAlreadyExists = xerrors.AlreadyExists("cbcschool already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid cbcschool input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("cbcschool conflict")
)

// Repository defines the contract for school persistence.
type Repository interface {
	Create(ctx context.Context, tenantID string, name string) (string, error)
	GetByID(ctx context.Context, id string) (*School, error)
	GetByTenantAndName(ctx context.Context, tenantID, name string) (*School, error)
	ListByTenantID(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error)
	Update(ctx context.Context, school SchoolUpdateFields) error
	Delete(ctx context.Context, id string) error
	OnboardingStatus(ctx context.Context, tenantID string) (*OnboardingStatus, error)
}

// School represents a CBC school record.
type School struct {
	ID        string    `db:"id"         json:"id"`
	TenantID  string    `db:"tenant_id"  json:"tenant_id"`
	Name      string    `db:"name"       json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// SchoolWithMemberCount extends School with member count data and active-school status.
type SchoolWithMemberCount struct {
	ID                   string    `db:"id"         json:"id"`
	TenantID             string    `db:"tenant_id"  json:"tenant_id"`
	Name                 string    `db:"name"       json:"name"`
	KnecSchoolCode       *string   `db:"knec_school_code" json:"knec_school_code,omitempty"`
	County               string    `db:"county"     json:"county"`
	SubCounty            string    `db:"sub_county" json:"sub_county"`
	Ward                 *string   `db:"ward"       json:"ward,omitempty"`
	SchoolType           string    `db:"school_type" json:"school_type"`
	IsActive             bool      `db:"is_active"  json:"is_active"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
	Admins               int       `db:"admins"     json:"admins"`
	Teachers             int       `db:"teachers"   json:"teachers"`
	Nurses               int       `db:"nurses"     json:"nurses"`
	Finance              int       `db:"finance"    json:"finance"`
	Parents              int       `db:"parents"    json:"parents"`
	Students             int       `db:"students"   json:"students"`
	IsMemberActiveSchool bool      `db:"is_member_active_school" json:"is_member_active_school"`
}

// SchoolUpdateFields holds fields that can be updated on a school.
type SchoolUpdateFields struct {
	ID             string
	Name           *string
	County         *string
	SubCounty      *string
	Ward           *string
	KnecSchoolCode *string
	NemisCode      *string
	SchoolType     *string
	IsActive       *bool
}

// OnboardingStatus represents the onboarding status of a tenant.
type OnboardingStatus struct {
	TenantID                   string
	ClassStreamsCreated        bool
	AcademicCalendarConfigured bool
	CurriculumInitialized      bool
	StaffInvited               bool
	StudentsEnrolled           bool
	IsOnboardingComplete       bool
}

// ListSchoolsResponse wraps a list of schools.
type ListSchoolsResponse struct {
	Items []SchoolWithMemberCount `json:"items"`
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	Limit int                     `json:"limit"`
}

// CurriculumSeeder seeds the CBC curriculum for a newly created school.
// Defined here as a consumer-side interface so cbcschools does not import
// the curriculum package directly (DDD boundary rule).
type CurriculumSeeder interface {
	SeedForSchool(ctx context.Context, tenantID, schoolID string) error
}

// UserSchoolEnroller creates a membership and sets the active school for a user.
// Defined here as a consumer-side interface to avoid importing the auth package
// directly (DDD boundary rule).
type UserSchoolEnroller interface {
	CreateMembership(ctx context.Context, userID, schoolID, tenantID, role string) error
	SetActiveSchool(ctx context.Context, userID, tenantID, schoolID string) error
	GetUserRoleInTenant(ctx context.Context, userID, tenantID string) (string, error)
}

// AcademicYearSeeder seeds the initial academic year and CBC terms for a
// newly created school. Defined here as a consumer-side interface so cbcschools
// does not import the academicyears package directly (DDD boundary rule).
type AcademicYearSeeder interface {
	SetupInitialYear(ctx context.Context, tenantID, schoolID, actorID string, now *time.Time) error
}
