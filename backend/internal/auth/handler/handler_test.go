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
	"somotracker/internal/platform/ratelimit"
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

func (m *mockStore) CreateUserSchool(ctx context.Context, orgID, userID, schoolID uuid.UUID) (storegen.UserSchool, error) {
	return storegen.UserSchool{}, nil
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

func (m *mockStore) InvalidateChallengesByEmail(ctx context.Context, emailNormalized string) error {
	return nil
}

// Ensure mockStore satisfies store.StoreInterface.
var _ store.StoreInterface = (*mockStore)(nil)

// mockTurnstile implements turnstile.Verifier for testing.
type mockTurnstile struct {
	result *turnstile.Result
	err    error
	siteKey string
}

func (m *mockTurnstile) Verify(ctx context.Context, token, remoteIP string) (*turnstile.Result, error) {
	return m.result, m.err
}

func (m *mockTurnstile) SiteKey() string {
	return m.siteKey
}

// mockCodeSender implements brevo.CodeSender.
type mockCodeSender struct {
	sendErr error
}

func (m *mockCodeSender) SendCode(ctx context.Context, email, code string) error {
	return m.sendErr
}

// mockLimiter implements ratelimit.Limiter for testing.
type mockLimiter struct {
	allowFunc func(ctx context.Context, key string, limit int, window time.Duration) (ratelimit.Decision, error)
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (ratelimit.Decision, error) {
	if m.allowFunc != nil {
		return m.allowFunc(ctx, key, limit, window)
	}
	return ratelimit.Decision{Allowed: true, Remaining: 10, Limit: 10}, nil
}

// Ensure mockLimiter satisfies ratelimit.Limiter.
var _ ratelimit.Limiter = (*mockLimiter)(nil)

// mockKeyBuilder implements ratelimit.KeyBuilder for testing.
type mockKeyBuilder struct {
	emailKeyFunc    func(string) string
	ipKeyFunc       func(string) string
	userKeyFunc     func(string) string
	challengeKeyFunc func(string) string
	globalKeyFunc   func() string
	domainKeyFunc   func(string) string
	tierPrefixFunc  func(string) string
}

func (m *mockKeyBuilder) EmailKey(email string) string {
	if m.emailKeyFunc != nil {
		return m.emailKeyFunc(email)
	}
	return "rl:otp_request:email:test"
}
func (m *mockKeyBuilder) IPKey(ip string) string {
	if m.ipKeyFunc != nil {
		return m.ipKeyFunc(ip)
	}
	return "rl:global:ip:test"
}
func (m *mockKeyBuilder) UserKey(userID string) string {
	if m.userKeyFunc != nil {
		return m.userKeyFunc(userID)
	}
	return "rl:global:user:test"
}
func (m *mockKeyBuilder) ChallengeKey(challengeID string) string {
	if m.challengeKeyFunc != nil {
		return m.challengeKeyFunc(challengeID)
	}
	return "rl:otp_verify:challenge:test"
}
func (m *mockKeyBuilder) GlobalKey() string {
	if m.globalKeyFunc != nil {
		return m.globalKeyFunc()
	}
	return "rl:otp_request:global:send_budget"
}
func (m *mockKeyBuilder) DomainKey(domain string) string {
	if m.domainKeyFunc != nil {
		return m.domainKeyFunc(domain)
	}
	return "rl:otp_request:domain:test"
}
func (m *mockKeyBuilder) TierPrefix(tier string) string {
	if m.tierPrefixFunc != nil {
		return m.tierPrefixFunc(tier)
	}
	return "rl:" + tier
}

// Ensure mockKeyBuilder satisfies ratelimit.KeyBuilder.
var _ ratelimit.KeyBuilder = (*mockKeyBuilder)(nil)

func newTestHandler(ts turnstile.Verifier, rl ratelimit.Limiter, kb ratelimit.KeyBuilder) *Handler {
	cfg := &config.Config{
		AppEnv:                        "development",
		SessionCookieName:             "session",
		EmailHMACKey:                  "test-hmac-key-32-bytes-long!!",
		TurnstileSecretKey:            "test-secret",
		TurnstileExpectedHost:         "example.com",
		RateLimitOTPRequestEmail:      3,
		RateLimitOTPRequestIP:         30,
		RateLimitOTPSendBudgetPerDay:  250,
		RateLimitOTPReservedReturningPct: 40,
		RateLimitOTPDomainVelocityPerHour: 20,
		RateLimitReturningUserTTLDays: 30,
		RateLimitDisposableDomains:    "",
		OTPVerifyMaxAttemptsPerChallenge: 5,
		RateLimitOTPVerifyEmailPerDay: 20,
		RateLimitOTPVerifyIPPerMin:    50,
		POWSecret:                     "test-pow-secret-32-bytes-long!!",
		POWBaseDifficulty:             4,
		POWChallengeTTLSeconds:        120,
		BotRiskStepUpThreshold:        50,
		BotRiskBlockThreshold:         85,
		BotRiskCriticalThreshold:      95,
		BFFDwellCookieName:            "__Host-dwell_ts",
		BFFDwellSubHumanMS:            800,
		BFFDwellStaleMS:               14400000,
		BotHoneypotFieldPrefix:        "hp_",
		RateLimitCriticalBudgetPct:    90,
		RateLimitOnboardUserPerMin:    5,
		RateLimitOnboardUserPerDay:    20,
		RateLimitOnboardIPPerMin:      50,
		RateLimitSessionUserPerMin:    30,
		RateLimitSessionIPPerMin:      100,
		RateLimitGlobalUserPerMin:     300,
		RateLimitGlobalIPPerMin:       600,
		RateLimitMetricsEnabled:       true,
		RateLimitAlertsEnabled:        true,
		RateLimitKeyHMACSecret:        "test-hmac-secret-32-bytes-long!!",
		BFFServiceSecret:              "test-bff-secret-32-bytes-long!!",
		TrustedProxyHops:              1,
		TrustedBFFCIDRs:               "",
	}
	log := logger.New("info")

	svc := service.New(&mockStore{}, &mockCodeSender{}, nil, cfg, log)

	return &Handler{
		svc:                  svc,
		cfg:                  cfg,
		log:                  log,
		turnstile:            ts,
		rateLimiter:          rl,
		keyBuilder:           kb,
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
	rl := &mockLimiter{}
	kb := &mockKeyBuilder{}
	ts := &mockTurnstile{
		siteKey: "test-site-key",
	}
	h := newTestHandler(ts, rl, kb)
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
		siteKey: "test-site-key",
	}
	rl := &mockLimiter{}
	kb := &mockKeyBuilder{}
	h := newTestHandler(ts, rl, kb)
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
		siteKey: "test-site-key",
	}
	rl := &mockLimiter{}
	kb := &mockKeyBuilder{}
	h := newTestHandler(ts, rl, kb)
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
		siteKey: "test-site-key",
	}
	rl := &mockLimiter{}
	kb := &mockKeyBuilder{}
	h := newTestHandler(ts, rl, kb)
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
		siteKey: "test-site-key",
	}
	rl := &mockLimiter{}
	kb := &mockKeyBuilder{}
	h := newTestHandler(ts, rl, kb)
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
	rl := &mockLimiter{}
	kb := &mockKeyBuilder{}
	h := newTestHandler(nil, rl, kb)
	app := setupApp(t, h)

	body := `{"email":"user@example.com"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestRequestOTP_RateLimitEmail tests that email rate limiting works.
func TestRequestOTP_RateLimitEmail(t *testing.T) {
	ts := &mockTurnstile{
		result: &turnstile.Result{Success: true, Hostname: "example.com"},
		err:    nil,
		siteKey: "test-site-key",
	}
	rl := &mockLimiter{
		allowFunc: func(ctx context.Context, key string, limit int, window time.Duration) (ratelimit.Decision, error) {
			// Simulate rate limit exceeded for email key
			if strings.Contains(key, "email") {
				return ratelimit.Decision{Allowed: false, Remaining: 0, Limit: 3, RetryAfter: time.Minute}, nil
			}
			return ratelimit.Decision{Allowed: true, Remaining: 10, Limit: 10}, nil
		},
	}
	kb := &mockKeyBuilder{}
	h := newTestHandler(ts, rl, kb)
	app := setupApp(t, h)

	body := `{"email":"user@example.com","turnstile_token":"valid-token"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	m := parseBody(t, resp)
	assert.Equal(t, "RATE_LIMITED", m["code"])
}

// TestRequestOTP_RateLimitIP tests that IP rate limiting works.
func TestRequestOTP_RateLimitIP(t *testing.T) {
	ts := &mockTurnstile{
		result: &turnstile.Result{Success: true, Hostname: "example.com"},
		err:    nil,
		siteKey: "test-site-key",
	}
	rl := &mockLimiter{
		allowFunc: func(ctx context.Context, key string, limit int, window time.Duration) (ratelimit.Decision, error) {
			// Simulate rate limit exceeded for IP key
			if strings.Contains(key, "ip") {
				return ratelimit.Decision{Allowed: false, Remaining: 0, Limit: 30, RetryAfter: time.Minute}, nil
			}
			return ratelimit.Decision{Allowed: true, Remaining: 10, Limit: 10}, nil
		},
	}
	kb := &mockKeyBuilder{}
	h := newTestHandler(ts, rl, kb)
	app := setupApp(t, h)

	body := `{"email":"user@example.com","turnstile_token":"valid-token"}`
	resp, err := postJSON(t, app, "/api/v1/auth/otp/request", body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	m := parseBody(t, resp)
	assert.Equal(t, "RATE_LIMITED", m["code"])
}