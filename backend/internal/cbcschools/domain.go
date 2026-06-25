package cbcschools

import (
	"context"
	"errors"
	"time"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("cbcschools not found")
	ErrAlreadyExists = errors.New("cbcschools already exists")
	ErrInvalidInput  = errors.New("invalid cbcschools input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("cbcschools conflict")
)

// Repository defines the contract for school persistence.
type Repository interface {
	Create(ctx context.Context, tenantID string, name string) (string, error)
	GetByID(ctx context.Context, id string) (*School, error)
}

// School represents a CBC school record.
type School struct {
	ID        string    `db:"id"         json:"id"`
	TenantID  string    `db:"tenant_id"  json:"tenant_id"`
	Name      string    `db:"name"       json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// SchoolMemberCounts represents a collection of member counts in a school.
type SchoolMemberCounts struct {
	SchoolID  string    `db:"school_id" json:"school_id"`
	Admins    int       `db:"admins" json:"admins"`
	Teachers  int       `db:"teachers" json:"teachers"`
	Nurses    int       `db:"nurses" json:"nurses"`
	Finance   int       `db:"finance" json:"finance"`
	Parents   int       `db:"parents" json:"parents"`
	Students  int       `db:"students" json:"students"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
