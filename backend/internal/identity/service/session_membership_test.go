package service

import (
	"context"
	"database/sql"
	"testing"

	storegen "somotracker/internal/identity/store/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	getSessionRow storegen.GetSessionWithMembershipsRow
	getSessionErr error
}

func (m *mockRepo) GetSessionWithMemberships(ctx context.Context, tokenHash []byte) (storegen.GetSessionWithMembershipsRow, error) {
	return m.getSessionRow, m.getSessionErr
}

// Implement all Repository methods as no-ops
func (m *mockRepo) GetPendingOnboardingByEmail(ctx context.Context, email string) (storegen.PendingOnboarding, error) {return storegen.PendingOnboarding{}, nil}
func (m *mockRepo) GetPendingOnboardingByID(ctx context.Context, id uuid.UUID) (storegen.PendingOnboarding, error) {return storegen.PendingOnboarding{}, nil}
func (m *mockRepo) UpsertPendingOnboarding(ctx context.Context, arg storegen.UpsertPendingOnboardingParams) (storegen.PendingOnboarding, error) {return storegen.PendingOnboarding{}, nil}
func (m *mockRepo) UpdatePendingOnboardingStep(ctx context.Context, arg storegen.UpdatePendingOnboardingStepParams) error {return nil}
func (m *mockRepo) UpdatePendingOnboardingProvisioning(ctx context.Context, id uuid.UUID, stytchOrgID, stytchMemberID uuid.UUID, status string) error {return nil}
func (m *mockRepo) CreateAppSession(ctx context.Context, arg storegen.CreateAppSessionParams) (storegen.AppSession, error) {return storegen.AppSession{}, nil}
func (m *mockRepo) GetAppSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.AppSession, error) {return storegen.AppSession{}, nil}
func (m *mockRepo) GetAppSessionByID(ctx context.Context, id uuid.UUID) (storegen.AppSession, error) {return storegen.AppSession{}, nil}
func (m *mockRepo) RevokeAppSession(ctx context.Context, id uuid.UUID) error {return nil}
func (m *mockRepo) CreateOnboardingTicket(ctx context.Context, arg storegen.CreateOnboardingTicketParams) (storegen.OnboardingTicket, error) {return storegen.OnboardingTicket{}, nil}
func (m *mockRepo) GetOnboardingTicketByHash(ctx context.Context, hash []byte) (storegen.OnboardingTicket, error) {return storegen.OnboardingTicket{}, nil}
func (m *mockRepo) MarkOnboardingTicketUsed(ctx context.Context, id uuid.UUID) error {return nil}
func (m *mockRepo) CreateTenant(ctx context.Context, arg storegen.CreateTenantParams) (storegen.Tenant, error) {return storegen.Tenant{}, nil}
func (m *mockRepo) GetTenantByStytchOrgID(ctx context.Context, stytchOrgID uuid.NullUUID) (storegen.Tenant, error) {return storegen.Tenant{}, nil}
func (m *mockRepo) CreateUser(ctx context.Context, arg storegen.CreateUserParams) (storegen.User, error) {return storegen.User{}, nil}
func (m *mockRepo) GetUserByStytchMemberID(ctx context.Context, stytchMemberID uuid.NullUUID) (storegen.User, error) {return storegen.User{}, nil}
func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (storegen.User, error) {return storegen.User{}, nil}
func (m *mockRepo) GetUserByEmailAndTenant(ctx context.Context, tenantID uuid.UUID, email string) (storegen.User, error) {return storegen.User{}, nil}
func (m *mockRepo) CreateSchool(ctx context.Context, arg storegen.CreateSchoolParams) (storegen.School, error) {return storegen.School{}, nil}
func (m *mockRepo) CreateMembership(ctx context.Context, arg storegen.CreateMembershipParams) (storegen.Membership, error) {return storegen.Membership{}, nil}
func (m *mockRepo) GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error) {return storegen.User{}, nil}
func (m *mockRepo) GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {return "", nil}
func (m *mockRepo) GetUserMemberships(ctx context.Context, userID, tenantID uuid.UUID) ([]MembershipContext, error) {return nil, nil}
func (m *mockRepo) GetGuardianStudents(ctx context.Context, guardianUserID, tenantID uuid.UUID) ([]uuid.UUID, error) {return nil, nil}
func (m *mockRepo) GetTeacherClasses(ctx context.Context, teacherUserID, tenantID uuid.UUID) ([]uuid.UUID, error) {return nil, nil}
func (m *mockRepo) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {return nil}

func TestValidateSessionWithMemberships_Success(t *testing.T) {

	tokenHash := []byte("hash")
	userID := uuid.New()
	tenantID := uuid.New()
	sessionID := uuid.New()
	schoolID := uuid.New()
	membershipID := uuid.New()

	mockRepo := &mockRepo{
		getSessionRow: storegen.GetSessionWithMembershipsRow{
			ID:             sessionID,
			UserID:         userID,
			TenantID:       tenantID,
			MembershipID:   uuid.NullUUID{UUID: membershipID, Valid: true},
			SchoolID:       uuid.NullUUID{UUID: schoolID, Valid: true},
			Role:           storegen.NullUserRole{UserRole: storegen.UserRoleTEACHER, Valid: true},
			IsActive:       sql.NullBool{Valid: true, Bool: true},
		},
	}

	svc := &Service{repo: mockRepo}

	ctx := context.Background()
	sess, mems, err := svc.ValidateSessionWithMemberships(ctx, tokenHash)
	require.NoError(t, err)
	require.NotNil(t, sess)
	require.Equal(t, userID, sess.UserID)
	require.Equal(t, tenantID, sess.TenantID)
	require.Len(t, mems, 1)
	require.Equal(t, "TEACHER", mems[0].Role)
	require.True(t, mems[0].IsActive)
}

func TestValidateSessionWithMemberships_NotFound(t *testing.T) {
	mockRepo := &mockRepo{
		getSessionErr: sql.ErrNoRows,
	}

	svc := &Service{repo: mockRepo}
	tokenHash := []byte("hash")

	ctx := context.Background()
	sess, mems, err := svc.ValidateSessionWithMemberships(ctx, tokenHash)
	require.ErrorIs(t, err, ErrSessionNotFound)
	require.Nil(t, sess)
	require.Nil(t, mems)
}
