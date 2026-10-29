package store

import (
	"context"
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

// Pending onboarding
func (s *Store) GetPendingOnboardingByEmail(ctx context.Context, email string) (storegen.PendingOnboarding, error) {
	return s.q.GetPendingOnboardingByEmail(ctx, email)
}

func (s *Store) GetPendingOnboardingByID(ctx context.Context, id uuid.UUID) (storegen.PendingOnboarding, error) {
	return s.q.GetPendingOnboardingByID(ctx, id)
}

func (s *Store) UpsertPendingOnboarding(ctx context.Context, arg storegen.UpsertPendingOnboardingParams) (storegen.PendingOnboarding, error) {
	return s.q.UpsertPendingOnboarding(ctx, arg)
}

func (s *Store) UpdatePendingOnboardingStep(ctx context.Context, arg storegen.UpdatePendingOnboardingStepParams) error {
	return s.q.UpdatePendingOnboardingStep(ctx, arg)
}

func (s *Store) UpdatePendingOnboardingProvisioning(ctx context.Context, id uuid.UUID, stytchOrgID, stytchMemberID uuid.UUID, status string) error {
	return s.q.UpdatePendingOnboardingProvisioning(ctx, storegen.UpdatePendingOnboardingProvisioningParams{
		ID:                  id,
		StytchOrganizationID: uuid.NullUUID{UUID: stytchOrgID, Valid: true},
		StytchMemberID:      uuid.NullUUID{UUID: stytchMemberID, Valid: true},
		Status:              status,
	})
}

// App session
func (s *Store) CreateAppSession(ctx context.Context, arg storegen.CreateAppSessionParams) (storegen.AppSession, error) {
	return s.q.CreateAppSession(ctx, arg)
}

func (s *Store) GetAppSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.AppSession, error) {
	return s.q.GetAppSessionByTokenHash(ctx, tokenHash)
}

func (s *Store) GetAppSessionByID(ctx context.Context, id uuid.UUID) (storegen.AppSession, error) {
	return s.q.GetAppSessionByID(ctx, id)
}

func (s *Store) RevokeAppSession(ctx context.Context, id uuid.UUID) error {
	return s.q.RevokeAppSession(ctx, id)
}

// Onboarding ticket
func (s *Store) CreateOnboardingTicket(ctx context.Context, arg storegen.CreateOnboardingTicketParams) (storegen.OnboardingTicket, error) {
	return s.q.CreateOnboardingTicket(ctx, arg)
}

func (s *Store) GetOnboardingTicketByHash(ctx context.Context, hash []byte) (storegen.OnboardingTicket, error) {
	return s.q.GetOnboardingTicketByHash(ctx, hash)
}

func (s *Store) MarkOnboardingTicketUsed(ctx context.Context, id uuid.UUID) error {
	return s.q.MarkOnboardingTicketUsed(ctx, id)
}

// Core domain
func (s *Store) CreateTenant(ctx context.Context, arg storegen.CreateTenantParams) (storegen.Tenant, error) {
	return s.q.CreateTenant(ctx, arg)
}

func (s *Store) GetTenantByID(ctx context.Context, id uuid.UUID) (storegen.Tenant, error) {
	return s.q.GetTenantByID(ctx, id)
}

func (s *Store) GetTenantByStytchOrgID(ctx context.Context, stytchOrgID uuid.NullUUID) (storegen.Tenant, error) {
	return s.q.GetTenantByStytchOrgID(ctx, stytchOrgID)
}

func (s *Store) CreateUser(ctx context.Context, arg storegen.CreateUserParams) (storegen.User, error) {
	return s.q.CreateUser(ctx, arg)
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error) {
	return s.q.GetUserByID(ctx, id)
}

func (s *Store) GetUserByEmailAndTenant(ctx context.Context, tenantID uuid.UUID, email string) (storegen.User, error) {
	return s.q.GetUserByEmailAndTenant(ctx, storegen.GetUserByEmailAndTenantParams{
		TenantID:        tenantID,
		EmailNormalized: email,
	})
}

func (s *Store) GetUserByStytchMemberID(ctx context.Context, stytchMemberID uuid.NullUUID) (storegen.User, error) {
	return s.q.GetUserByStytchMemberID(ctx, stytchMemberID)
}

func (s *Store) CreateSchool(ctx context.Context, arg storegen.CreateSchoolParams) (storegen.School, error) {
	return s.q.CreateSchool(ctx, arg)
}

func (s *Store) GetSchoolByID(ctx context.Context, id uuid.UUID) (storegen.School, error) {
	return s.q.GetSchoolByID(ctx, id)
}

func (s *Store) CreateMembership(ctx context.Context, arg storegen.CreateMembershipParams) (storegen.Membership, error) {
	return s.q.CreateMembership(ctx, arg)
}

func (s *Store) GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]storegen.Membership, error) {
	return s.q.GetMembershipsByUserID(ctx, userID)
}

func (s *Store) GetMembershipByUserAndSchool(ctx context.Context, userID, schoolID uuid.UUID) (storegen.Membership, error) {
	return s.q.GetMembershipByUserAndSchool(ctx, storegen.GetMembershipByUserAndSchoolParams{
		UserID:   userID,
		SchoolID: schoolID,
	})
}

func (s *Store) CreatePKCECode(ctx context.Context, pendingOnboardingID uuid.UUID, codeVerifierHash []byte, expiresAt time.Time) error {
	return s.q.CreatePKCECode(ctx, storegen.CreatePKCECodeParams{
		PendingOnboardingID: pendingOnboardingID,
		CodeVerifierHash:    codeVerifierHash,
		ExpiresAt:           expiresAt,
	})
}

func (s *Store) GetPKCECode(ctx context.Context, pendingOnboardingID uuid.UUID) (storegen.PkceCode, error) {
	return s.q.GetPKCECode(ctx, pendingOnboardingID)
}

func (s *Store) DeletePKCECode(ctx context.Context, pendingOnboardingID uuid.UUID) error {
	return s.q.DeletePKCECode(ctx, pendingOnboardingID)
}

func (s *Store) GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {
	return s.q.GetTenantMembershipRole(ctx, storegen.GetTenantMembershipRoleParams{
		ID:       userID,
		TenantID: tenantID,
	})
}