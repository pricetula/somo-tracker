package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"somotracker/internal/auth/store"
	storegen "somotracker/internal/auth/store/gen"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// mockBrevoSender implements brevo.CodeSender for testing.
type mockBrevoSender struct {
	mock.Mock
}

func (m *mockBrevoSender) SendCode(ctx context.Context, email, code string) error {
	args := m.Called(ctx, email, code)
	return args.Error(0)
}

// mockStore implements store.StoreInterface for testing.
type mockStore struct {
	mock.Mock
}

func (m *mockStore) CreateLoginChallenge(ctx context.Context, challengeID uuid.UUID, emailNormalized string, codeMac []byte, expiresAt time.Time) (storegen.LoginChallenge, error) {
	args := m.Called(ctx, challengeID, emailNormalized, codeMac, expiresAt)
	return args.Get(0).(storegen.LoginChallenge), args.Error(1)
}

func (m *mockStore) CreateEmailOutbox(ctx context.Context, challengeID uuid.UUID, encryptedPayload []byte) (storegen.EmailOutbox, error) {
	args := m.Called(ctx, challengeID, encryptedPayload)
	return args.Get(0).(storegen.EmailOutbox), args.Error(1)
}

func (m *mockStore) GetLoginChallengeByID(ctx context.Context, id uuid.UUID) (storegen.LoginChallenge, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(storegen.LoginChallenge), args.Error(1)
}

func (m *mockStore) MarkChallengeUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockStore) GetUserByNormalizedEmail(ctx context.Context, email string) (storegen.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(storegen.User), args.Error(1)
}

func (m *mockStore) CreateOrganization(ctx context.Context, name string) (storegen.Organization, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(storegen.Organization), args.Error(1)
}

func (m *mockStore) CreateUser(ctx context.Context, orgID uuid.UUID, email, normalizedEmail, name string) (storegen.User, error) {
	args := m.Called(ctx, orgID, email, normalizedEmail, name)
	return args.Get(0).(storegen.User), args.Error(1)
}

func (m *mockStore) CreateSchool(ctx context.Context, orgID uuid.UUID, name string, countryID, educationSystemID uuid.NullUUID) (storegen.School, error) {
	args := m.Called(ctx, orgID, name, countryID, educationSystemID)
	return args.Get(0).(storegen.School), args.Error(1)
}

func (m *mockStore) CreateUserSchool(ctx context.Context, orgID, userID, schoolID uuid.UUID) (storegen.UserSchool, error) {
	args := m.Called(ctx, orgID, userID, schoolID)
	return args.Get(0).(storegen.UserSchool), args.Error(1)
}

func (m *mockStore) UpdateOrganization(ctx context.Context, orgID uuid.UUID, name string) error {
	args := m.Called(ctx, orgID, name)
	return args.Error(0)
}

func (m *mockStore) UpdateUser(ctx context.Context, userID uuid.UUID, name string) error {
	args := m.Called(ctx, userID, name)
	return args.Error(0)
}

func (m *mockStore) UpdateSessionType(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockStore) GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.Session, error) {
	args := m.Called(ctx, tokenHash)
	return args.Get(0).(storegen.Session), args.Error(1)
}

func (m *mockStore) GetSchoolsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.School, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]storegen.School), args.Error(1)
}

func (m *mockStore) CreateSession(ctx context.Context, tokenHash []byte, userID, orgID uuid.UUID, activeSchoolID uuid.NullUUID, sessionType string, expiresAt time.Time) (storegen.Session, error) {
	args := m.Called(ctx, tokenHash, userID, orgID, activeSchoolID, sessionType, expiresAt)
	return args.Get(0).(storegen.Session), args.Error(1)
}

func (m *mockStore) InvalidateChallengesByEmail(ctx context.Context, emailNormalized string) error {
	args := m.Called(ctx, emailNormalized)
	return args.Error(0)
}

func newTestService(s store.StoreInterface, cs *mockBrevoSender, cfg *config.Config) *Service {
	return &Service{store: s, codeSender: cs, redis: nil, cfg: cfg, log: logger.Default()}
}

func testConfig() *config.Config {
	return &config.Config{
		EmailHMACKey: "test-hmac-key-32-bytes-long!!",
	}
}

func anyUUID() interface{} {
	return mock.MatchedBy(func(id uuid.UUID) bool { return id != uuid.Nil })
}

func anyCodeMac() interface{} {
	return mock.MatchedBy(func(b []byte) bool { return len(b) == 32 })
}

func anyTime() interface{} {
	return mock.MatchedBy(func(t time.Time) bool { return !t.IsZero() })
}

func TestRequestOTP_Success(t *testing.T) {
	ctx := context.Background()
	email := "user@example.com"
	challengeID := uuid.New()

	store := &mockStore{}
	store.On("InvalidateChallengesByEmail", ctx, email).
		Return(nil).
		Once()
	store.On("CreateLoginChallenge", ctx, anyUUID(), email, anyCodeMac(), anyTime()).
		Return(storegen.LoginChallenge{ID: challengeID, EmailNormalized: email}, nil).
		Once()
	store.On("CreateEmailOutbox", ctx, challengeID, mock.Anything).
		Return(storegen.EmailOutbox{}, nil).
		Once()

	sender := &mockBrevoSender{}
	sender.On("SendCode", mock.Anything, email, mock.Anything).Return(nil).Once()

	svc := newTestService(store, sender, testConfig())

	resp, err := svc.RequestOTP(ctx, RequestOTPRequest{Email: email})

	require.NoError(t, err)
	assert.True(t, resp.OK)
	assert.Equal(t, challengeID.String(), resp.ChallengeID)

	// Brevo is fire-and-forget in a goroutine; give it a moment
	time.Sleep(50 * time.Millisecond)
	sender.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestRequestOTP_InvalidEmail(t *testing.T) {
	ctx := context.Background()

	store := &mockStore{}
	sender := &mockBrevoSender{}
	svc := newTestService(store, sender, testConfig())

	_, err := svc.RequestOTP(ctx, RequestOTPRequest{Email: "not-an-email"})

	require.Error(t, err)
	assert.Equal(t, apperr.INVALID_EMAIL, err.(*apperr.Error).Code)

	store.AssertExpectations(t)
	sender.AssertExpectations(t)
}

func TestRequestOTP_StoreError(t *testing.T) {
	ctx := context.Background()
	email := "user@example.com"

	store := &mockStore{}
	store.On("InvalidateChallengesByEmail", ctx, email).
		Return(nil).
		Once()
	store.On("CreateLoginChallenge", ctx, anyUUID(), email, anyCodeMac(), anyTime()).
		Return(storegen.LoginChallenge{}, errors.New("db down")).
		Once()

	sender := &mockBrevoSender{}
	svc := newTestService(store, sender, testConfig())

	_, err := svc.RequestOTP(ctx, RequestOTPRequest{Email: email})

	require.Error(t, err)
	assert.Equal(t, apperr.INTERNAL_ERROR, err.(*apperr.Error).Code)

	store.AssertExpectations(t)
	sender.AssertExpectations(t)
}

func TestVerifyOTP_Success(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	userID := uuid.New()
	orgID := uuid.New()
	code := "123456"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "user@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		AttemptCount:    0,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{},
	}

	user := storegen.User{
		ID:              userID,
		OrganizationID:  orgID,
		Email:           "user@example.com",
		NormalizedEmail: "user@example.com",
		Status:          "ACTIVE",
	}

	schools := []storegen.School{{ID: uuid.New(), Name: "Test School"}}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()
	store.On("MarkChallengeUsed", ctx, challengeID).Return(nil).Once()
	store.On("GetUserByNormalizedEmail", ctx, "user@example.com").Return(user, nil).Once()
	store.On("GetSchoolsByUser", ctx, userID).Return(schools, nil).Once()
	store.On("CreateSession", ctx, mock.Anything, userID, orgID, mock.Anything, mock.Anything, mock.Anything).
		Return(storegen.Session{}, nil).Once()

	svc := newTestService(store, &mockBrevoSender{}, cfg)

	resp, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.NoError(t, err)
	assert.True(t, resp.OK)
	assert.NotEmpty(t, resp.SessionToken)

	store.AssertExpectations(t)
}

func TestVerifyOTP_InvalidCode(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"
	wrongCode := "000000"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "user@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		AttemptCount:    0,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{},
	}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()

	svc := newTestService(store, &mockBrevoSender{}, cfg)

	_, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: wrongCode})

	require.Error(t, err)
	assert.Equal(t, apperr.INVALID_TOKEN, err.(*apperr.Error).Code)
	assert.Equal(t, "invalid code", err.(*apperr.Error).Message)

	store.AssertExpectations(t)
}

func TestVerifyOTP_ExpiredChallenge(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "user@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(-1 * time.Minute), // expired
		AttemptCount:    0,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{},
	}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()

	svc := newTestService(store, &mockBrevoSender{}, cfg)

	_, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.Error(t, err)
	assert.Equal(t, apperr.INVALID_TOKEN, err.(*apperr.Error).Code)
	assert.Equal(t, "challenge expired", err.(*apperr.Error).Message)

	store.AssertExpectations(t)
}

func TestVerifyOTP_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "user@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		AttemptCount:    5,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{},
	}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()

	svc := newTestService(store, &mockBrevoSender{}, cfg)

	_, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.Error(t, err)
	assert.Equal(t, apperr.INVALID_TOKEN, err.(*apperr.Error).Code)
	assert.Equal(t, "max attempts exceeded", err.(*apperr.Error).Message)

	store.AssertExpectations(t)
}

func TestVerifyOTP_AlreadyUsed(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "user@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		AttemptCount:    0,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{Valid: true, Time: time.Now()},
	}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()

	svc := newTestService(store, &mockBrevoSender{}, cfg)

	_, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.Error(t, err)
	assert.Equal(t, apperr.INVALID_TOKEN, err.(*apperr.Error).Code)
	assert.Equal(t, "challenge already used", err.(*apperr.Error).Message)

	store.AssertExpectations(t)
}

func TestVerifyOTP_NoUserFound_CreatesAccount(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "newuser@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		AttemptCount:    0,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{},
	}

	newOrgID := uuid.New()
	newUserID := uuid.New()

	org := storegen.Organization{
		ID:     newOrgID,
		Name:   fmt.Sprintf("org-%s", challengeID.String()[:8]),
		Status: "ACTIVE",
	}
	user := storegen.User{
		ID:              newUserID,
		OrganizationID:  newOrgID,
		Email:           "newuser@example.com",
		NormalizedEmail: "newuser@example.com",
		Name:            sql.NullString{},
		Status:          "ACTIVE",
	}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()
	store.On("MarkChallengeUsed", ctx, challengeID).Return(nil).Once()
	store.On("GetUserByNormalizedEmail", ctx, "newuser@example.com").
		Return(storegen.User{}, errNoRows).Once()
	store.On("CreateOrganization", ctx, fmt.Sprintf("org-%s", challengeID.String()[:8])).
		Return(org, nil).Once()
	store.On("CreateSchool", ctx, newOrgID, "School", uuid.NullUUID{}, uuid.NullUUID{}).Return(storegen.School{}, nil).Once()
	store.On("CreateUser", ctx, newOrgID, "newuser@example.com", "newuser@example.com", "").
		Return(user, nil).Once()
	store.On("GetSchoolsByUser", ctx, newUserID).Return([]storegen.School{}, nil).Once()
	store.On("CreateSession", ctx, mock.Anything, newUserID, newOrgID, mock.Anything, mock.Anything, mock.Anything).
		Return(storegen.Session{}, nil).Once()

	sender := &mockBrevoSender{}
	svc := newTestService(store, sender, cfg)

	resp, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.NoError(t, err)
	assert.True(t, resp.OK)
	assert.NotEmpty(t, resp.SessionToken)
	assert.True(t, resp.NeedsOnboarding)

	store.AssertExpectations(t)
	sender.AssertExpectations(t)
}

func TestVerifyOTP_StoreError(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).
		Return(storegen.LoginChallenge{}, errors.New("db down")).Once()

	svc := newTestService(store, &mockBrevoSender{}, testConfig())

	_, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.Error(t, err)
	assert.Equal(t, apperr.INTERNAL_ERROR, err.(*apperr.Error).Code)

	store.AssertExpectations(t)
}

func TestVerifyOTP_CreatesNewUser(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()
	code := "123456"

	cfg := testConfig()
	codeMac := computeHMAC(cfg.EmailHMACKey, challengeID, code)

	challenge := storegen.LoginChallenge{
		ID:              challengeID,
		EmailNormalized: "newuser@example.com",
		CodeMac:         codeMac,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		AttemptCount:    0,
		MaxAttempts:     5,
		UsedAt:          sql.NullTime{},
	}

	newOrgID := uuid.New()
	newUserID := uuid.New()

	org := storegen.Organization{
		ID:     newOrgID,
		Name:   "newuser",
		Status: "ACTIVE",
	}
	user := storegen.User{
		ID:              newUserID,
		OrganizationID:  newOrgID,
		Email:           "newuser@example.com",
		NormalizedEmail: "newuser@example.com",
		Name:            sql.NullString{},
		Status:          "ACTIVE",
	}

	store := &mockStore{}
	store.On("GetLoginChallengeByID", ctx, challengeID).Return(challenge, nil).Once()
	store.On("MarkChallengeUsed", ctx, challengeID).Return(nil).Once()
	store.On("GetUserByNormalizedEmail", ctx, "newuser@example.com").
		Return(storegen.User{}, errNoRows).Once()
	store.On("CreateOrganization", ctx, mock.Anything).
		Return(org, nil).Once()
	store.On("CreateSchool", ctx, newOrgID, "School", uuid.NullUUID{}, uuid.NullUUID{}).Return(storegen.School{}, nil).Once()
	store.On("CreateUser", ctx, newOrgID, "newuser@example.com", "newuser@example.com", "").
		Return(user, nil).Once()
	store.On("GetSchoolsByUser", ctx, newUserID).Return([]storegen.School{}, nil).Once()
	store.On("CreateSession", ctx, mock.Anything, newUserID, newOrgID, mock.Anything, mock.Anything, mock.Anything).
		Return(storegen.Session{}, nil).Once()

	sender := &mockBrevoSender{}
	svc := newTestService(store, sender, cfg)

	resp, err := svc.VerifyOTP(ctx, VerifyOTPRequest{ChallengeID: challengeID, Code: code})

	require.NoError(t, err)
	assert.True(t, resp.OK)
	assert.NotEmpty(t, resp.SessionToken)

	store.AssertExpectations(t)
	sender.AssertExpectations(t)
}

func TestOnboard_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orgID := uuid.New()

	store := &mockStore{}
	store.On("UpdateOrganization", ctx, orgID, "My Org").Return(nil).Once()
	store.On("UpdateUser", ctx, userID, "John Doe").Return(nil).Once()
	store.On("CreateSchool", ctx, orgID, "My School", uuid.NullUUID{}, uuid.NullUUID{}).Return(storegen.School{ID: uuid.New()}, nil).Once()
	store.On("CreateUserSchool", ctx, orgID, userID, mock.MatchedBy(func(id uuid.UUID) bool { return id != uuid.Nil })).Return(storegen.UserSchool{}, nil).Once()
	store.On("UpdateSessionType", ctx, userID).Return(nil).Once()

	svc := newTestService(store, &mockBrevoSender{}, testConfig())

	err := svc.Onboard(ctx, userID, orgID, "My Org", "My School", "John Doe", uuid.NullUUID{}, uuid.NullUUID{})

	require.NoError(t, err)
	store.AssertExpectations(t)
}

func TestOnboard_StoreError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orgID := uuid.New()

	store := &mockStore{}
	store.On("UpdateOrganization", ctx, orgID, "My Org").Return(errors.New("db down")).Once()

	svc := newTestService(store, &mockBrevoSender{}, testConfig())

	err := svc.Onboard(ctx, userID, orgID, "My Org", "My School", "John Doe", uuid.NullUUID{}, uuid.NullUUID{})

	require.Error(t, err)
	assert.Equal(t, apperr.INTERNAL_ERROR, err.(*apperr.Error).Code)

	store.AssertExpectations(t)
}
