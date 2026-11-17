package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/identity/store/gen"
)

// Store implements Repository for identity domain.
// SQL lives in queries/*.sql, generated to gen/ by sqlc (BE-009).
type Store struct {
	q *storegen.Queries
}

// New creates store.
func New(q *storegen.Queries) *Store {
	return &Store{q: q}
}

// Conversion helpers
func toPendingOnboarding(row storegen.GetPendingOnboardingByEmailRow) storegen.PendingOnboarding {
	return storegen.PendingOnboarding{
		ID:                      row.ID,
		VerifiedEmailNormalized: row.VerifiedEmailNormalized,
		Status:                  row.Status,
		CurrentStep:             row.CurrentStep,
		FormData:                row.FormData,
		StytchOrganizationID:    row.StytchOrganizationID,
		StytchMemberID:          row.StytchMemberID,
		TenantID:                row.TenantID,
		SchoolName:              row.SchoolName,
		IdempotencyKey:          row.IdempotencyKey,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		LastActivityAt:          row.LastActivityAt,
		DeletedAt:               sql.NullTime{},
	}
}

func toPendingOnboardingByID(row storegen.GetPendingOnboardingByIDRow) storegen.PendingOnboarding {
	return storegen.PendingOnboarding{
		ID:                      row.ID,
		VerifiedEmailNormalized: row.VerifiedEmailNormalized,
		Status:                  row.Status,
		CurrentStep:             row.CurrentStep,
		FormData:                row.FormData,
		StytchOrganizationID:    row.StytchOrganizationID,
		StytchMemberID:          row.StytchMemberID,
		TenantID:                row.TenantID,
		SchoolName:              row.SchoolName,
		IdempotencyKey:          row.IdempotencyKey,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		LastActivityAt:          row.LastActivityAt,
		DeletedAt:               sql.NullTime{},
	}
}

func toPendingOnboardingUpsert(row storegen.UpsertPendingOnboardingRow) storegen.PendingOnboarding {
	return storegen.PendingOnboarding{
		ID:                      row.ID,
		VerifiedEmailNormalized: row.VerifiedEmailNormalized,
		Status:                  row.Status,
		CurrentStep:             row.CurrentStep,
		FormData:                row.FormData,
		StytchOrganizationID:    row.StytchOrganizationID,
		StytchMemberID:          row.StytchMemberID,
		TenantID:                row.TenantID,
		SchoolName:              row.SchoolName,
		IdempotencyKey:          row.IdempotencyKey,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		LastActivityAt:          row.LastActivityAt,
		DeletedAt:               sql.NullTime{},
	}
}

func toAppSession(row storegen.CreateAppSessionRow) storegen.AppSession {
	return storegen.AppSession{
		ID:                     row.ID,
		UserID:                 row.UserID,
		TenantID:               row.TenantID,
		TokenHash:              row.TokenHash,
		StytchMemberSessionID:  row.StytchMemberSessionID,
		CreatedAt:              row.CreatedAt,
		ExpiresAt:              row.ExpiresAt,
		RevokedAt:              row.RevokedAt,
		DeletedAt:              sql.NullTime{},
		LastSeenAt:             row.LastSeenAt,
	}
}

func toAppSessionByToken(row storegen.GetAppSessionByTokenHashRow) storegen.AppSession {
	return storegen.AppSession{
		ID:                     row.ID,
		UserID:                 row.UserID,
		TenantID:               row.TenantID,
		TokenHash:              row.TokenHash,
		StytchMemberSessionID:  row.StytchMemberSessionID,
		CreatedAt:              row.CreatedAt,
		ExpiresAt:              row.ExpiresAt,
		RevokedAt:              row.RevokedAt,
		DeletedAt:              sql.NullTime{},
		LastSeenAt:             row.LastSeenAt,
	}
}

func toAppSessionByID(row storegen.GetAppSessionByIDRow) storegen.AppSession {
	return storegen.AppSession{
		ID:                     row.ID,
		UserID:                 row.UserID,
		TenantID:               row.TenantID,
		TokenHash:              row.TokenHash,
		StytchMemberSessionID:  row.StytchMemberSessionID,
		CreatedAt:              row.CreatedAt,
		ExpiresAt:              row.ExpiresAt,
		RevokedAt:              row.RevokedAt,
		DeletedAt:              sql.NullTime{},
		LastSeenAt:             row.LastSeenAt,
	}
}

func toOnboardingTicketCreate(row storegen.CreateOnboardingTicketRow) storegen.OnboardingTicket {
	return storegen.OnboardingTicket{
		ID:                  row.ID,
		PendingOnboardingID: row.PendingOnboardingID,
		TicketHash:          row.TicketHash,
		IstCiphertext:       row.IstCiphertext,
		UsedAt:              row.UsedAt,
		ExpiresAt:           row.ExpiresAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           time.Time{},
		DeletedAt:           sql.NullTime{},
		IstNonce:            nil,
		IstKeyVersion:       0,
	}
}

func toOnboardingTicket(row storegen.GetOnboardingTicketByHashRow) storegen.OnboardingTicket {
	return storegen.OnboardingTicket{
		ID:                  row.ID,
		PendingOnboardingID: row.PendingOnboardingID,
		TicketHash:          row.TicketHash,
		IstCiphertext:       row.IstCiphertext,
		UsedAt:              row.UsedAt,
		ExpiresAt:           row.ExpiresAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           time.Time{},
		DeletedAt:           sql.NullTime{},
		IstNonce:            nil,
		IstKeyVersion:       0,
	}
}

func toUserByID(row storegen.GetUserByIDRow) storegen.User {
	return storegen.User{
		ID:              row.ID,
		TenantID:        row.TenantID,
		EmailNormalized: row.EmailNormalized,
		StytchMemberID:  row.StytchMemberID,
		Name:            row.Name,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       sql.NullTime{},
	}
}

// Pending onboarding
func (s *Store) GetPendingOnboardingByEmail(ctx context.Context, email string) (storegen.PendingOnboarding, error) {
	row, err := s.q.GetPendingOnboardingByEmail(ctx, email)
	return toPendingOnboarding(row), err
}

func (s *Store) GetPendingOnboardingByID(ctx context.Context, id uuid.UUID) (storegen.PendingOnboarding, error) {
	row, err := s.q.GetPendingOnboardingByID(ctx, id)
	return toPendingOnboardingByID(row), err
}

func (s *Store) UpsertPendingOnboarding(ctx context.Context, arg storegen.UpsertPendingOnboardingParams) (storegen.PendingOnboarding, error) {
	row, err := s.q.UpsertPendingOnboarding(ctx, arg)
	return toPendingOnboardingUpsert(row), err
}

func (s *Store) UpdatePendingOnboardingStep(ctx context.Context, arg storegen.UpdatePendingOnboardingStepParams) error {
	return s.q.UpdatePendingOnboardingStep(ctx, arg)
}

// App session
func (s *Store) CreateAppSession(ctx context.Context, arg storegen.CreateAppSessionParams) (storegen.AppSession, error) {
	row, err := s.q.CreateAppSession(ctx, arg)
	return toAppSession(row), err
}

func (s *Store) GetAppSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.AppSession, error) {
	row, err := s.q.GetAppSessionByTokenHash(ctx, tokenHash)
	return toAppSessionByToken(row), err
}

func (s *Store) GetAppSessionByID(ctx context.Context, id uuid.UUID) (storegen.AppSession, error) {
	row, err := s.q.GetAppSessionByID(ctx, id)
	return toAppSessionByID(row), err
}

func (s *Store) RevokeAppSession(ctx context.Context, id uuid.UUID) error {
	return s.q.RevokeAppSession(ctx, id)
}

func (s *Store) UpdateSessionLastSeen(ctx context.Context, sessionID uuid.UUID, idleThreshold time.Duration) error {
	return s.q.UpdateSessionLastSeen(ctx, storegen.UpdateSessionLastSeenParams{
		ID:           sessionID,
		Column2:      idleThreshold,
	})
}

func (s *Store) GetSessionWithMemberships(ctx context.Context, tokenHash []byte) (storegen.GetSessionWithMembershipsRow, error) {
	return s.q.GetSessionWithMemberships(ctx, tokenHash)
}

// Onboarding ticket
func (s *Store) CreateOnboardingTicket(ctx context.Context, arg storegen.CreateOnboardingTicketParams) (storegen.OnboardingTicket, error) {
	row, err := s.q.CreateOnboardingTicket(ctx, arg)
	return toOnboardingTicketCreate(row), err
}

func (s *Store) GetOnboardingTicketByHash(ctx context.Context, hash []byte) (storegen.OnboardingTicket, error) {
	row, err := s.q.GetOnboardingTicketByHash(ctx, hash)
	return toOnboardingTicket(row), err
}

// Session
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error) {
	row, err := s.q.GetUserByID(ctx, id)
	return toUserByID(row), err
}

func (s *Store) GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {
	role, err := s.q.GetTenantMembershipRole(ctx, storegen.GetTenantMembershipRoleParams{
		ID:       userID,
		TenantID: tenantID,
	})
	return string(role), err
}
