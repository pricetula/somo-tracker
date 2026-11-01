package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
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

	svc := service.New(repo, mockStytch, db)
	cfg := &config.Config{
		Env: "development",
	}
	h := New(svc, cfg, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/onboarding/complete", h.ProvisionTenant)

	normalizedEmail := "handler-test@example.com"
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

	mockStytch.exchangeSession = &stytch.DiscoveryExchangeResponse{
		SessionToken: "fake-stytch-session",
	}

	body := map[string]string{
		"school_name":     "Kitengela Handler Test",
		"idempotency_key": "idem-handler-" + uuid.New().String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(bodyBytes))
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
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("no Set-Cookie headers")
	}

	foundSession := false
	foundOnboardingCleared := false
	for _, c := range cookies {
		if contains(c, "session_token_development=") && contains(c, "HttpOnly") && contains(c, "SameSite=Lax") {
			foundSession = true
		}
		if contains(c, "onboarding=") && contains(c, "max-age=0") {
			foundOnboardingCleared = true
		}
	}

	if !foundSession {
		t.Errorf("session cookie not set: %v", cookies)
	}
	if !foundOnboardingCleared {
		t.Errorf("onboarding cookie not cleared: %v", cookies)
	}

	// Verify tenant created
	tenant, err := repo.GetTenantByStytchOrgID(ctx, uuid.NullUUID{UUID: uuid.MustParse(mockStytch.orgID), Valid: true})
	if err != nil {
		t.Fatalf("tenant not found: %v", err)
	}
	if tenant.Name != "Kitengela Handler Test" {
		t.Errorf("tenant name = %q", tenant.Name)
	}

	_ = cleanupHandlerTestData(ctx, db)
}

type mockStytchHandler struct {
	orgID           string
	memberID        string
	exchangeSession *stytch.DiscoveryExchangeResponse
}

func (m *mockStytchHandler) SendDiscoveryMagicLink(ctx context.Context, email, loginURL, pkceCodeChallenge string) error {
	return nil
}
func (m *mockStytchHandler) AuthenticateDiscovery(ctx context.Context, token, pkceCodeVerifier string) (*stytch.DiscoveryAuthResponse, error) {
	return &stytch.DiscoveryAuthResponse{Email: "test@example.com", IntermediateSessionToken: "ist"}, nil
}
func (m *mockStytchHandler) ExchangeDiscoverySession(ctx context.Context, intermediateSessionToken string) (*stytch.DiscoveryExchangeResponse, error) {
	if m.exchangeSession != nil {
		return m.exchangeSession, nil
	}
	return &stytch.DiscoveryExchangeResponse{SessionToken: "token"}, nil
}
func (m *mockStytchHandler) CreateOrganizationAndMember(ctx context.Context, name, email string) (*stytch.OrgMemberResponse, error) {
	return &stytch.OrgMemberResponse{OrganizationID: m.orgID, MemberID: m.memberID}, nil
}
func (m *mockStytchHandler) RevokeMemberSession(ctx context.Context, memberID, sessionID string) error {
	return nil
}

func cleanupHandlerTestData(ctx context.Context, db *sql.DB) error {
	_, _ = db.ExecContext(ctx, `DELETE FROM membership WHERE user_id IN (SELECT id FROM "user" WHERE email_normalized LIKE 'handler-%'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM school WHERE tenant_id IN (SELECT id FROM tenant WHERE name LIKE 'Kitengela Handler%'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM "user" WHERE email_normalized LIKE 'handler-%'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM tenant WHERE name LIKE 'Kitengela Handler%'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM onboarding_ticket WHERE pending_onboarding_id IN (SELECT id FROM pending_onboarding WHERE verified_email_normalized LIKE 'handler-%')`)
	_, _ = db.ExecContext(ctx, `DELETE FROM pending_onboarding WHERE verified_email_normalized LIKE 'handler-%'`)
	return nil
}
