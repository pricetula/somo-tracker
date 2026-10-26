package service

// Service defines business logic for identity domain (BE-002).
type Service struct {
	repo Repository
}

// Repository is domain repository interface, defined by consumer (BE-005).
type Repository interface {
	// TODO: declare methods matching store queries
}

// New creates service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}
