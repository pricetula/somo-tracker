package store

// Store implements Repository for identity domain.
// SQL lives in queries/*.sql, generated to gen/ by sqlc (BE-009).
type Store struct {
	// TODO: pgx pool or Tx
}

// New creates store.
func New() *Store {
	return &Store{}
}
