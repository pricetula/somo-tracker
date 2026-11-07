package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/identity/service"
	"somotracker/internal/identity/store"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/stytch"
)

func TestProvisionTenant_Handler_CompleteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := sql.Open("pgx", "postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available, skipping integration test: %v", err)
	}

	ctx := context.Background()
	_ = cleanupHandlerTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	mockStytch := &mockStytchHandler{
		orgID:    uuid.New().String(),
		memberID: uuid.New().String(),
	}

	svc, err := service.New(repo, mockStytch, db, "")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cfg := &config.Config{
		Env: "development",
	}
	h := New(svc, cfg, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/onboarding/complete", h.ProvisionTenant)

	normalizedEmail := "handler-test-" + uuid.New().String() + "@example.com"
	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "handler-ticket-" + uuid.New().String()
	ticketHash := sha256.Sum256([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash[:],
		EncryptedIst:        []byte("test-ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Test provisioning
	reqBody := map[string]string{
		"school_name":      "Handler Test School " + uuid.New().String(),
		"idempotency_key":  "idem-" + uuid.New().String(),
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "onboarding",
		Value: ticketToken,
	})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want %d, body: %s", resp.StatusCode, http.StatusOK, string(bodyBytes))
	}

	var result ProvisionTenantResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if !result.Success {
		t.Error("expected success=true")
	}

	// Verify session cookie is set
	cookies := resp.Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_token_development" {
			found = true
			break
		}
	}
	if !found {
		t.Error("session cookie not set")
	}
}

func TestProvisionTenant_StytchFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := sql.Open("pgx", "postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available, skipping integration test: %v", err)
	}

	ctx := context.Background()
	_ = cleanupHandlerTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	mockStytch := &mockStytchHandler{
		failCreateOrg: true,
	}

	svc, err := service.New(repo, mockStytch, db, "")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cfg := &config.Config{
		Env: "development",
	}
	h := New(svc, cfg, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/onboarding/complete", h.ProvisionTenant)

	normalizedEmail := "handler-stytch-fail-" + uuid.New().String() + "@example.com"
	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "handler-ticket-" + uuid.New().String()
	ticketHash := sha256.Sum256([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash[:],
		EncryptedIst:        []byte("test-ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	reqBody := map[string]string{
		"school_name":      "Stytch Fail School " + uuid.New().String(),
		"idempotency_key":  "idem-" + uuid.New().String(),
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "onboarding",
		Value: ticketToken,
	})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestProvisionTenant_CompletedPendingBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := sql.Open("pgx", "postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available, skipping integration test: %v", err)
	}

	ctx := context.Background()
	_ = cleanupHandlerTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	mockStytch := &mockStytchHandler{
		orgID:    uuid.New().String(),
		memberID: uuid.New().String(),
	}

	svc, err := service.New(repo, mockStytch, db, "")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cfg := &config.Config{
		Env: "development",
	}
	h := New(svc, cfg, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/onboarding/complete", h.ProvisionTenant)

	normalizedEmail := "handler-completed-" + uuid.New().String() + "@example.com"
	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "completed",
		CurrentStep:             "complete",
		FormData:                []byte(`{}`),
		TenantID:                uuid.NullUUID{UUID: uuid.New(), Valid: true},
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "handler-ticket-" + uuid.New().String()
	ticketHash := sha256.Sum256([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash[:],
		EncryptedIst:        []byte("test-ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	reqBody := map[string]string{
		"school_name":      "Completed Pending School " + uuid.New().String(),
		"idempotency_key":  "idem-" + uuid.New().String(),
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "onboarding",
		Value: ticketToken,
	})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	// Should fail because pending is already completed
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestProvisionTenant_ExistingUserBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := sql.Open("pgx", "postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available, skipping integration test: %v", err)
	}

	ctx := context.Background()
	_ = cleanupHandlerTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	mockStytch := &mockStytchHandler{
		orgID:    uuid.New().String(),
		memberID: uuid.New().String(),
	}

	svc, err := service.New(repo, mockStytch, db, "")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cfg := &config.Config{
		Env: "development",
	}
	h := New(svc, cfg, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/onboarding/complete", h.ProvisionTenant)

	// First, create a user with this email
	normalizedEmail := "handler-existing-" + uuid.New().String() + "@example.com"
	tenant, err := repo.CreateTenant(ctx, storegen.CreateTenantParams{
		Name:                   "Existing Tenant " + uuid.New().String(),
		StytchOrganizationID:   uuid.NullUUID{UUID: uuid.New(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	_, err = repo.CreateUser(ctx, storegen.CreateUserParams{
		TenantID:           tenant.ID,
		EmailNormalized:    normalizedEmail,
		StytchMemberID:     uuid.NullUUID{UUID: uuid.New(), Valid: true},
		Name:               "Existing User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Now try to provision with same email
	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "handler-ticket-" + uuid.New().String()
	ticketHash := sha256.Sum256([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash[:],
		EncryptedIst:        []byte("test-ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	reqBody := map[string]string{
		"school_name":      "Existing User School " + uuid.New().String(),
		"idempotency_key":  "idem-" + uuid.New().String(),
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "onboarding",
		Value: ticketToken,
	})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	// Should fail because user already exists
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// mockStytchHandler implements stytch.Client for testing
type mockStytchHandler struct {
	orgID        string
	memberID     string
	failCreateOrg bool
}

func (m *mockStytchHandler) SendDiscoveryMagicLink(ctx context.Context, email, loginURL, challenge string) error {
	return nil
}

func (m *mockStytchHandler) AuthenticateDiscovery(ctx context.Context, token, pkceCodeVerifier string) (*stytch.DiscoveryAuthResponse, error) {
	return &stytch.DiscoveryAuthResponse{
		Email:           "test@example.com",
		IntermediateSessionToken: "test-ist",
		DiscoveredOrganizations: []stytch.DiscoveredOrg{},
	}, nil
}

func (m *mockStytchHandler) CreateOrganizationAndMember(ctx context.Context, orgName, email string) (*stytch.OrgMemberResponse, error) {
	if m.failCreateOrg {
		return nil, errors.New("stytch error")
	}
	return &stytch.OrgMemberResponse{
		OrganizationID: m.orgID,
		MemberID:       m.memberID,
	}, nil
}

func (m *mockStytchHandler) ExchangeDiscoverySession(ctx context.Context, ist string) (*stytch.DiscoveryExchangeResponse, error) {
	return nil, nil
}

func (m *mockStytchHandler) RevokeMemberSession(ctx context.Context, memberID, sessionID string) error {
	return nil
}

// testErrorHandler returns a test error handler
func testErrorHandler() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return c.Status(apperr.Status(ae.Code)).JSON(fiber.Map{
				"code":       ae.Code,
				"message":    ae.Message,
				"errors":     ae.Fields,
				"request_id": c.Locals("request_id"),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":       "INTERNAL_ERROR",
			"message":    "internal server error",
			"request_id": c.Locals("request_id"),
		})
	}
}

func cleanupHandlerTestData(ctx context.Context, db *sql.DB) error {
	// Clean up test data – execute statements individually for pgx compatibility
	stmts := []string{
		`DELETE FROM onboarding_ticket WHERE pending_onboarding_id IN (SELECT id FROM pending_onboarding WHERE verified_email_normalized LIKE 'handler-%')`,
		`DELETE FROM pending_onboarding WHERE verified_email_normalized LIKE 'handler-%'`,
		`DELETE FROM app_session WHERE user_id IN (SELECT id FROM users WHERE email_normalized LIKE 'handler-%')`,
		`DELETE FROM memberships WHERE user_id IN (SELECT id FROM users WHERE email_normalized LIKE 'handler-%')`,
		`DELETE FROM users WHERE email_normalized LIKE 'handler-%'`,
		`DELETE FROM schools WHERE tenant_id IN (SELECT id FROM tenants WHERE name LIKE 'Handler%')`,
		`DELETE FROM tenants WHERE name LIKE 'Handler%' OR name LIKE 'Existing%'`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}