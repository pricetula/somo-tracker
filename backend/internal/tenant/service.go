package tenant

// Service contains business logic for tenant operations.
type Service struct {
	repo *SqlcRepository
}

// NewService creates a new Service.
// Accepts the concrete *SqlcRepository to avoid fx ambiguous-binding errors
// during scaffolding. Refactor to interface injection once the first real
// method is added.
func NewService(repo *SqlcRepository) *Service {
	return &Service{repo: repo}
}
