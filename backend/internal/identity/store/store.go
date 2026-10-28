package store

import store "somotracker/internal/identity/store/gen"

// Store implements Repository for identity domain.
// SQL lives in queries/*.sql, generated to gen/ by sqlc (BE-009).
type Store struct {
	q *store.Queries
}

// New creates store.
func New(q *store.Queries) *Store {
	return &Store{q: q}
}
