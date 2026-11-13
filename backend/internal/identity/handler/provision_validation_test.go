package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/identity/service"
	"somotracker/internal/identity/store"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/stytch"
)

// mockStytchHandlerForValidation implements stytch.Client for validation tests
type mockStytchHandlerForValidation struct{}

func (m *mockStytchHandlerForValidation) SendDiscoveryMagicLink(ctx context.Context, email, loginURL, challenge string) error {
	return nil
}

func (m *mockStytchHandlerForValidation) AuthenticateDiscovery(ctx context.Context, token, pkceCodeVerifier string) (*stytch.DiscoveryAuthResponse, error) {
	return nil, nil
}

func (m *mockStytchHandlerForValidation) CreateOrganizationAndMember(ctx context.Context, orgName, email string) (*stytch.OrgMemberResponse, error) {
	return nil, nil
}

func (m *mockStytchHandlerForValidation) ExchangeDiscoverySession(ctx context.Context, ist string) (*stytch.DiscoveryExchangeResponse, error) {
	return nil, nil
}

func (m *mockStytchHandlerForValidation) RevokeMemberSession(ctx context.Context, memberID, sessionID string) error {
	return nil
}

// TestProvisionTenant_Handler_RequiresCountryID verifies that the handler
// rejects requests missing country_id with a 400 error. This prevents
// provisioning without a country, which would result in a school with nil
// country_id (violating DB NOT NULL constraints).
func TestProvisionTenant_Handler_RequiresCountryID(t *testing.T) {
	// Use a real repo but mock Stytch - we won't reach Stytch because validation fails first
	db, err := sql.Open("pgx", "postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable")
	if err != nil {
		t.Skipf("cannot open DB: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available: %v", err)
	}

	ctx := context.Background()
	_ = cleanupHandlerTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	mockStytch := &mockStytchHandlerForValidation{}
	svc, err := service.New(repo, mockStytch, db, "")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cfg := &config.Config{Env: "development"}
	h := New(svc, cfg, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/onboarding/complete", h.ProvisionTenant)

	// Missing country_id
	reqBody := map[string]string{
		"school_name":      "Test School " + uuid.New().String(),
		"idempotency_key":  "idem-" + uuid.New().String(),
		"education_system_id": uuid.New().String(),
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "onboarding", Value: "ticket-token"})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (missing country_id should be 400)", resp.StatusCode, http.StatusBadRequest)
	}

	// Missing education_system_id
	reqBody2 := map[string]string{
		"school_name":      "Test School " + uuid.New().String(),
		"idempotency_key":  "idem-" + uuid.New().String(),
		"country_id":       uuid.New().String(),
	}
	reqBytes2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes2))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: "onboarding", Value: "ticket-token"})

	resp2, err := app.Test(req2, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (missing education_system_id should be 400)", resp2.StatusCode, http.StatusBadRequest)
	}

	// Invalid UUID for country_id
	reqBody3 := map[string]string{
		"school_name":       "Test School " + uuid.New().String(),
		"idempotency_key":   "idem-" + uuid.New().String(),
		"country_id":        "not-a-uuid",
		"education_system_id": uuid.New().String(),
	}
	reqBytes3, _ := json.Marshal(reqBody3)

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes3))
	req3.Header.Set("Content-Type", "application/json")
	req3.AddCookie(&http.Cookie{Name: "onboarding", Value: "ticket-token"})

	resp3, err := app.Test(req3, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp3.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (invalid country_id should be 400)", resp3.StatusCode, http.StatusBadRequest)
	}

	// All fields present should get past validation (will fail later at Stytch/ticket, but not validation)
	reqBody4 := map[string]string{
		"school_name":        "Test School " + uuid.New().String(),
		"idempotency_key":    "idem-" + uuid.New().String(),
		"country_id":         uuid.New().String(),
		"education_system_id": uuid.New().String(),
	}
	reqBytes4, _ := json.Marshal(reqBody4)

	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(reqBytes4))
	req4.Header.Set("Content-Type", "application/json")
	req4.AddCookie(&http.Cookie{Name: "onboarding", Value: "ticket-token"})

	resp4, err := app.Test(req4, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	// Should NOT be 400 with a missing-field validation error
	// It may be 400 with INVALID_TICKET (ticket is fake) or 500 for other reasons,
	// but not a validation error about missing country_id/education_system_id
	if resp4.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp4.Body)
		bodyStr := string(body)
		// If it's 400, it should be about the invalid ticket, not missing fields
		if !strings.Contains(bodyStr, "INVALID_TICKET") {
			t.Errorf("status = %d, body = %s - all fields present but got unexpected 400", resp4.StatusCode, bodyStr)
		}
	}
}
