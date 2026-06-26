package cbcschools

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
	ErrNotFound      = fmt.Errorf("cbcschools not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("cbcschools already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid cbcschools input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("cbcschools conflict: %w", middleware.ErrConflict)
)

// Repository defines the contract for school persistence.
type Repository interface {
	Create(ctx context.Context, tenantID string, name string) (string, error)
	GetByID(ctx context.Context, id string) (*School, error)
	ListByTenantID(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error)
	Update(ctx context.Context, school SchoolUpdateFields) error
	Delete(ctx context.Context, id string) error
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

// ListSchoolsResponse wraps a list of schools.
type ListSchoolsResponse struct {
	Schools []SchoolWithMemberCount `json:"schools"`
	Total   int                     `json:"total"`
}
