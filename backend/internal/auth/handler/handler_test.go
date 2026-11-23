package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storegen "somotracker/internal/auth/store/gen"
	"somotracker/internal/auth/store"
	"somotracker/internal/auth/service"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/httpserver"
	"somotracker/internal/platform/logger"
	"somotracker/internal/platform/turnstile"
)

// mockStore implements store.StoreInterface for handler tests.
type mockStore struct {
	createChallengeErr error
	createOutboxErr    error
	lastEmail          string
}

func (m *mockStore) CreateLoginChallenge(ctx context.Context, challengeID uuid.UUID, emailNormalized string, codeMac []byte, expiresAt time.Time) (storegen.LoginChallenge, error) {
	m.lastEmail = emailNormalized
	if m.createChallengeErr != nil {
		return storegen.LoginChallenge{}, m.createChallengeErr
	}
	return storegen.LoginChallenge{ID: challengeID, EmailNormalized: emailNormalized}, nil
}

func (m *mockStore) CreateEmailOutbox(ctx context.Context, challengeID uuid.UUID, encryptedPayload []byte) (storegen.EmailOutbox, error) {
	if m.createOutboxErr != nil {
		return storegen.EmailOutbox{}, m.createOutboxErr
	}
	return storegen.EmailOutbox{}, nil
}

func (m *mockStore) GetLoginChallengeByID(ctx context.Context, id uuid.UUID) (storegen.LoginChallenge, error) {
	return storegen.LoginChallenge{}, nil
}

func (m *mockStore) MarkChallengeUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockStore) GetUserByNormalizedEmail(ctx context.Context, email string) (storegen.User, error) {
	return storegen.User{}, nil
}

func (m *mockStore) CreateOrganization(ctx context.Context, name string) (storegen.Organization, error) {
	return storegen.Organization{}, nil
}

func (m *mockStore) CreateUser(ctx context.Context, orgID uuid.UUID, email, normalizedEmail, name string) (storegen.User, error) {
	return storegen.User{}, nil
}

func (m *mockStore) CreateSchool(ctx context.Context, orgID uuid.UUID, name string) (storegen.School, error) {
	return storegen.School{}, nil
}

func (m *mockStore) UpdateOrganization(ctx context.Context, orgID uuid.UUID, name string) error {
	return nil
}

func (m *mockStore) UpdateUser(ctx context.Context, userID uuid.UUID, name string) error {
	return nil
}

func (m *mockStore) UpdateSessionType(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (m *mockStore) GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.Session, error) {
	return storegen.Session{}, nil
}

func (m *mockStore) GetSchoolsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.School, error) {
	return nil, nil
}

func (m *mockStore) CreateSession(ctx context.Context, tokenHash []byte, userID, orgID uuid.UUID, activeSchoolID uuid.NullUUID, sessionType string, expiresAt time.Time) (storegen.Session, error) {
	return storegen.Session{}, nil
}

// Ensure mockStore satisfies store.StoreInterface.
var _ store.StoreInterface = (*mockStore)(nil)

// mockTurnstile implements turnstile.Verifier for testing.
type mockTurnstile struct {
	result *turnstile.Result
	err    error
}

func (m *mockTurnstile) Verify(ctx context.Context, token, remoteIP string) (*turnstile.Result, error) {
	return m.result, m.err
}

// mockCodeSender implements brevo.CodeSender.
type mockCodeSender struct {
	sendErr error
}

func (m *mockCodeSender) SendCode(ctx context.Context, email, code string) error {
	return m.sendErr
}

func newTestHandler(ts turnstile.Verifier) *Handler {
	cfg := &config.Config{
		AppEnv:                "development",
		SessionCookieName:     "session",
		EmailHMACKey:          "test-hmac-key-32-bytes-long!!",
		TurnstileSecretKey:    "test-secret",
		TurnstileExpectedHost: "example.com",
	}
	log := logger.New("info")

	svc := service.New(&mockStore{}, &mockCodeSender{}, cfg, log)

	return &Handler{
		svc:                  svc,
		cfg:                  cfg,
		log:                  log,
		turnstile:            ts,
		sessionCookieName:    "session",
		onboardingCookieName: "needs_onboarding",
	}
}

// setupApp creates a fiber app with only the otp/request route mounted.
func setupApp(t *testing.T, h *Handler) *fiber.App {
	t.Helper()
	app := httpserver.New("development").App()
	app.Post("/api/v1/auth/otp/request", h.RequestOTP)
	return app
}

func postJSON(t *testing.T, app *fiber.App, path, body string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return app.Test(req)
}

func parseBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)
	return m
}

func TestRequestOTP_MissingTurnstileToken(t *testing.T) {
	h := newTestHandler(nil)
	app := setupApp(t, h)

	body := `{"email":"user@example.com"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	m := parseBody(t, resp)
	assert.Equal(t, "INVALID_REQUEST", m["code"])
}

func TestRequestOTP_TurnstileFailedVerification(t *testing.T) {
	ts := &mockTurnstile{
		result: &turnstile.Result{Success: false, ErrorCodes: []string{"invalid-input-response"}},
		err:    turnstile.ErrVerificationFailed,
	}
	h := newTestHandler(ts)
	app := setupApp(t, h)

	body := `{"email":"user@example.com","turnstile_token":"bad-token"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	m := parseBody(t, resp)
	assert.Equal(t, "FORBIDDEN", m["code"])
}

func TestRequestOTP_TurnstileUpstreamFailure(t *testing.T) {
	ts := &mockTurnstile{
		result: nil,
		err:    turnstile.ErrUpstream,
	}
	h := newTestHandler(ts)
	app := setupApp(t, h)

	body := `{"email":"user@example.com","turnstile_token":"token"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	m := parseBody(t, resp)
	assert.Equal(t, "BAD_GATEWAY", m["code"])
}

func TestRequestOTP_Success(t *testing.T) {
	ts := &mockTurnstile{
		result: &turnstile.Result{Success: true, Hostname: "example.com"},
		err:    nil,
	}
	h := newTestHandler(ts)
	app := setupApp(t, h)

	body := `{"email":"user@example.com","turnstile_token":"valid-token"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := parseBody(t, resp)
	assert.Equal(t, true, m["ok"])
	assert.Equal(t, "If an account exists for this email, a code has been sent.", m["message"])
	assert.NotEmpty(t, m["challenge_id"])
}

func TestRequestOTP_InvalidEmail(t *testing.T) {
	ts := &mockTurnstile{
		result: &turnstile.Result{Success: true, Hostname: "example.com"},
		err:    nil,
	}
	h := newTestHandler(ts)
	app := setupApp(t, h)

	body := `{"email":"not-an-email","turnstile_token":"valid-token"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRequestOTP_NoTurnstileVerifier_StillRequiresToken(t *testing.T) {
	// When no turnstile verifier is configured, the handler still requires
	// the token field to be present (empty token => 400).
	h := newTestHandler(nil)
	app := setupApp(t, h)

	body := `{"email":"user@example.com"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
