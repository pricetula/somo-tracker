package store

import (
	"context"

	"github.com/google/uuid"
	store "somotracker/internal/identity/store/gen"
)

// Store implements Repository for identity domain.
// SQL lives in queries/*.sql, generated to gen/ by sqlc (BE-009).
type Store struct {
	q *store.Queries
}

// New creates store.
func New(q *store.Queries) *Store {
	return &Store{q: q}
}

// Pending onboarding
func (s *Store) GetPendingOnboardingByEmail(ctx context.Context, email string) (store.PendingOnboarding, error) {
	return s.q.GetPendingOnboardingByEmail(ctx, email)
}

func (s *Store) UpsertPendingOnboarding(ctx context.Context, arg store.UpsertPendingOnboardingParams) (store.PendingOnboarding, error) {
	return s.q.UpsertPendingOnboarding(ctx, arg)
}

func (s *Store) UpdatePendingOnboardingStep(ctx context.Context, arg store.UpdatePendingOnboardingStepParams) error {
	return s.q.UpdatePendingOnboardingStep(ctx, arg)
}

// App session
func (s *Store) CreateAppSession(ctx context.Context, arg store.CreateAppSessionParams) (store.AppSession, error) {
	return s.q.CreateAppSession(ctx, arg)
}

func (s *Store) GetAppSessionByTokenHash(ctx context.Context, tokenHash []byte) (store.AppSession, error) {
	return s.q.GetAppSessionByTokenHash(ctx, tokenHash)
}

func (s *Store) RevokeAppSession(ctx context.Context, id uuid.UUID) error {
	return s.q.RevokeAppSession(ctx, id)
}

// Onboarding ticket
func (s *Store) CreateOnboardingTicket(ctx context.Context, arg store.CreateOnboardingTicketParams) (store.OnboardingTicket, error) {
	return s.q.CreateOnboardingTicket(ctx, arg)
}

func (s *Store) GetOnboardingTicketByHash(ctx context.Context, hash []byte) (store.OnboardingTicket, error) {
	return s.q.GetOnboardingTicketByHash(ctx, hash)
}

func (s *Store) MarkOnboardingTicketUsed(ctx context.Context, id uuid.UUID) error {
	return s.q.MarkOnboardingTicketUsed(ctx, id)
}
