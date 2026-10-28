package service

// Service defines business logic for identity domain (BE-002).
type Service struct {
	repo Repository
}

// Repository is domain repository interface, defined by consumer (BE-005).
type Repository interface {
	// TODO: define methods matching store queries
	// GetPendingOnboardingByEmail, UpsertPendingOnboarding, UpdatePendingOnboardingStep
	// CreateAppSession, GetAppSessionByTokenHash, RevokeAppSession
	// CreateOnboardingTicket, GetOnboardingTicketByHash, MarkOnboardingTicketUsed
}

// New creates service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}
