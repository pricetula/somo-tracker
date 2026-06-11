package tenant

import "time"

// Tenant represents an educational institution or organisation using the platform.
type Tenant struct {
	ID        string    `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Slug      string    `db:"slug"       json:"slug"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
