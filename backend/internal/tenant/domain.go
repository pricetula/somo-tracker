package tenant

import (
	"context"
	"errors"
	"time"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("tenant not found")
	ErrAlreadyExists = errors.New("tenant already exists")
	ErrInvalidInput  = errors.New("invalid tenant input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("tenant conflict")
)

// Repository defines the contract for tenant persistence.
type Repository interface {
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	Create(ctx context.Context, name, slug string) (*Tenant, error)
	GetByID(ctx context.Context, id string) (*Tenant, error)
}

// Tenant represents an educational institution or organisation using the platform.
type Tenant struct {
	ID        string    `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Slug      string    `db:"slug"       json:"slug"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
