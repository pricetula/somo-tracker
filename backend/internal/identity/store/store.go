package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/identity/service"
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

func toTenantCommon(id uuid.UUID, name string, stytchOrgID uuid.NullUUID, tenantStatus storegen.TenantStatus, createdAt, updatedAt time.Time) storegen.Tenant {
	return storegen.Tenant{
		ID:                     id,
		Name:                   name,
		StytchOrganizationID:   stytchOrgID,
		TenantStatus:           tenantStatus,
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
		DeletedAt:              sql.NullTime{},
	}
}

func toTenantCreate(row storegen.CreateTenantRow) storegen.Tenant {
	return toTenantCommon(row.ID, row.Name, row.StytchOrganizationID, row.TenantStatus, row.CreatedAt, row.UpdatedAt)
}

func toTenantByID(row storegen.GetTenantByIDRow) storegen.Tenant {
	return toTenantCommon(row.ID, row.Name, row.StytchOrganizationID, row.TenantStatus, row.CreatedAt, row.UpdatedAt)
}

func toTenantByStytchOrg(row storegen.GetTenantByStytchOrgIDRow) storegen.Tenant {
	return toTenantCommon(row.ID, row.Name, row.StytchOrganizationID, row.TenantStatus, row.CreatedAt, row.UpdatedAt)
}

func toUserCommon(id, tenantID uuid.UUID, emailNormalized string, stytchMemberID uuid.NullUUID, name string, createdAt, updatedAt time.Time) storegen.User {
	return storegen.User{
		ID:              id,
		TenantID:        tenantID,
		EmailNormalized: emailNormalized,
		StytchMemberID:  stytchMemberID,
		Name:            name,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       sql.NullTime{},
	}
}

func toUserCreate(row storegen.CreateUserRow) storegen.User {
	return toUserCommon(row.ID, row.TenantID, row.EmailNormalized, row.StytchMemberID, row.Name, row.CreatedAt, row.UpdatedAt)
}

func toUserByID(row storegen.GetUserByIDRow) storegen.User {
	return toUserCommon(row.ID, row.TenantID, row.EmailNormalized, row.StytchMemberID, row.Name, row.CreatedAt, row.UpdatedAt)
}

func toUserByEmailAndTenant(row storegen.GetUserByEmailAndTenantRow) storegen.User {
	return toUserCommon(row.ID, row.TenantID, row.EmailNormalized, row.StytchMemberID, row.Name, row.CreatedAt, row.UpdatedAt)
}

func toUserByStytchMember(row storegen.GetUserByStytchMemberIDRow) storegen.User {
	return toUserCommon(row.ID, row.TenantID, row.EmailNormalized, row.StytchMemberID, row.Name, row.CreatedAt, row.UpdatedAt)
}

func toUserByEmail(row storegen.GetUserByEmailRow) storegen.User {
	return toUserCommon(row.ID, row.TenantID, row.EmailNormalized, row.StytchMemberID, row.Name, row.CreatedAt, row.UpdatedAt)
}

func toSchoolCommon(id, tenantID uuid.UUID, name, code string, createdAt, updatedAt time.Time) storegen.School {
	return storegen.School{
		ID:                id,
		TenantID:          tenantID,
		Name:              name,
		Code:              code,
		CountryID:         uuid.Nil,
		EducationSystemID: uuid.Nil,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		DeletedAt:         sql.NullTime{},
	}
}

func toSchoolCreate(row storegen.CreateSchoolRow) storegen.School {
	return toSchoolCommon(row.ID, row.TenantID, row.Name, row.Code, row.CreatedAt, row.UpdatedAt)
}

func toSchoolByID(row storegen.GetSchoolByIDRow) storegen.School {
	return toSchoolCommon(row.ID, row.TenantID, row.Name, row.Code, row.CreatedAt, row.UpdatedAt)
}

func toMembershipCommon(id, userID, schoolID uuid.UUID, role storegen.UserRole, createdAt time.Time) storegen.Membership {
	return storegen.Membership{
		ID:        id,
		UserID:    userID,
		SchoolID:  schoolID,
		Role:      role,
		IsActive:  true,
		CreatedAt: createdAt,
		DeletedAt: sql.NullTime{},
	}
}

func toMembershipCreate(row storegen.CreateMembershipRow) storegen.Membership {
	return toMembershipCommon(row.ID, row.UserID, row.SchoolID, row.Role, row.CreatedAt)
}

func toMembershipByUser(row storegen.GetMembershipsByUserIDRow) storegen.Membership {
	return toMembershipCommon(row.ID, row.UserID, row.SchoolID, row.Role, row.CreatedAt)
}

func toMembershipByUserAndSchool(row storegen.GetMembershipByUserAndSchoolRow) storegen.Membership {
	return toMembershipCommon(row.ID, row.UserID, row.SchoolID, row.Role, row.CreatedAt)
}

func toPkceCode(row storegen.GetPKCECodeRow) storegen.PkceCode {
	return storegen.PkceCode{
		PendingOnboardingID: row.PendingOnboardingID,
		CodeVerifierHash:    row.CodeVerifierHash,
		ExpiresAt:           row.ExpiresAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.CreatedAt,
		DeletedAt:           sql.NullTime{},
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

func (s *Store) UpdatePendingOnboardingProvisioning(ctx context.Context, id uuid.UUID, stytchOrgID, stytchMemberID uuid.UUID, status string) error {
	return s.q.UpdatePendingOnboardingProvisioning(ctx, storegen.UpdatePendingOnboardingProvisioningParams{
		ID:                     id,
		StytchOrganizationID:   uuid.NullUUID{UUID: stytchOrgID, Valid: true},
		StytchMemberID:         uuid.NullUUID{UUID: stytchMemberID, Valid: true},
		Status:                 storegen.PendingOnboardingStatus(status),
	})
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

func (s *Store) MarkOnboardingTicketUsed(ctx context.Context, id uuid.UUID) error {
	return s.q.MarkOnboardingTicketUsed(ctx, id)
}

// Core domain
func (s *Store) CreateTenant(ctx context.Context, arg storegen.CreateTenantParams) (storegen.Tenant, error) {
	row, err := s.q.CreateTenant(ctx, arg)
	return toTenantCreate(row), err
}

func (s *Store) GetTenantByID(ctx context.Context, id uuid.UUID) (storegen.Tenant, error) {
	row, err := s.q.GetTenantByID(ctx, id)
	return toTenantByID(row), err
}

func (s *Store) GetTenantByStytchOrgID(ctx context.Context, stytchOrgID uuid.NullUUID) (storegen.Tenant, error) {
	row, err := s.q.GetTenantByStytchOrgID(ctx, stytchOrgID)
	return toTenantByStytchOrg(row), err
}

func (s *Store) CreateUser(ctx context.Context, arg storegen.CreateUserParams) (storegen.User, error) {
	row, err := s.q.CreateUser(ctx, arg)
	return toUserCreate(row), err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error) {
	row, err := s.q.GetUserByID(ctx, id)
	return toUserByID(row), err
}

func (s *Store) GetUserByEmailAndTenant(ctx context.Context, tenantID uuid.UUID, email string) (storegen.User, error) {
	row, err := s.q.GetUserByEmailAndTenant(ctx, storegen.GetUserByEmailAndTenantParams{
		TenantID:        tenantID,
		EmailNormalized: email,
	})
	return toUserByEmailAndTenant(row), err
}

func (s *Store) GetUserByStytchMemberID(ctx context.Context, stytchMemberID uuid.NullUUID) (storegen.User, error) {
	row, err := s.q.GetUserByStytchMemberID(ctx, stytchMemberID)
	return toUserByStytchMember(row), err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (storegen.User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	return toUserByEmail(row), err
}

func (s *Store) CreateSchool(ctx context.Context, arg storegen.CreateSchoolParams) (storegen.School, error) {
	row, err := s.q.CreateSchool(ctx, arg)
	return toSchoolCreate(row), err
}

func (s *Store) GetSchoolByID(ctx context.Context, id uuid.UUID) (storegen.School, error) {
	row, err := s.q.GetSchoolByID(ctx, id)
	return toSchoolByID(row), err
}

func (s *Store) CreateMembership(ctx context.Context, arg storegen.CreateMembershipParams) (storegen.Membership, error) {
	row, err := s.q.CreateMembership(ctx, arg)
	return toMembershipCreate(row), err
}

func (s *Store) GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]storegen.Membership, error) {
	rows, err := s.q.GetMembershipsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]storegen.Membership, len(rows))
	for i, row := range rows {
		result[i] = toMembershipByUser(row)
	}
	return result, nil
}

func (s *Store) GetMembershipByUserAndSchool(ctx context.Context, userID, schoolID uuid.UUID) (storegen.Membership, error) {
	row, err := s.q.GetMembershipByUserAndSchool(ctx, storegen.GetMembershipByUserAndSchoolParams{
		UserID:   userID,
		SchoolID: schoolID,
	})
	return toMembershipByUserAndSchool(row), err
}

func (s *Store) CreatePKCECode(ctx context.Context, pendingOnboardingID uuid.UUID, codeVerifierHash []byte, expiresAt time.Time) error {
	return s.q.CreatePKCECode(ctx, storegen.CreatePKCECodeParams{
		PendingOnboardingID: pendingOnboardingID,
		CodeVerifierHash:    codeVerifierHash,
		ExpiresAt:           expiresAt,
	})
}

func (s *Store) GetPKCECode(ctx context.Context, pendingOnboardingID uuid.UUID) (storegen.PkceCode, error) {
	row, err := s.q.GetPKCECode(ctx, pendingOnboardingID)
	return toPkceCode(row), err
}

func (s *Store) DeletePKCECode(ctx context.Context, pendingOnboardingID uuid.UUID) error {
	return s.q.DeletePKCECode(ctx, pendingOnboardingID)
}

func (s *Store) GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {
	role, err := s.q.GetTenantMembershipRole(ctx, storegen.GetTenantMembershipRoleParams{
		ID:       userID,
		TenantID: tenantID,
	})
	return string(role), err
}

// RBAC support - placeholder implementations
// TODO: Add custom SQL queries for these
func (s *Store) GetUserMemberships(ctx context.Context, userID, tenantID uuid.UUID) ([]service.MembershipContext, error) {
	return []service.MembershipContext{}, nil
}

func (s *Store) GetGuardianStudents(ctx context.Context, guardianUserID, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func (s *Store) GetTeacherClasses(ctx context.Context, teacherUserID, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.q.RevokeAppSession(ctx, sessionID)
}

func (s *Store) ListCountries(ctx context.Context) ([]storegen.Country, error) {
	return s.q.ListCountries(ctx)
}

func (s *Store) ListEducationSystems(ctx context.Context, countryID uuid.UUID) ([]storegen.EducationSystem, error) {
	return s.q.ListEducationSystems(ctx, countryID)
}

func (s *Store) ListSchoolsByTenant(ctx context.Context, tenantID uuid.UUID) ([]storegen.School, error) {
	rows, err := s.q.ListSchoolsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]storegen.School, len(rows))
	for i, row := range rows {
		result[i] = storegen.School{
			ID:                row.ID,
			TenantID:          row.TenantID,
			Name:              row.Name,
			Code:              row.Code,
			CountryID:         row.CountryID,
			EducationSystemID: row.EducationSystemID,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		}
	}
	return result, nil
}

func (s *Store) SetMembershipActive(ctx context.Context, userID uuid.UUID, membershipID uuid.UUID) error {
	return s.q.SetMembershipActive(ctx, storegen.SetMembershipActiveParams{
		UserID: userID,
		ID:     membershipID,
	})
}
