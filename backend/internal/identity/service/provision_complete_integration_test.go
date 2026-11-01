package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/identity/store"
	"somotracker/internal/platform/stytch"
)

// TestProvisionTenant_CompleteFlow validates the full onboarding complete flow
// from ticket validation through tenant provisioning to session creation.
func TestProvisionTenant_CompleteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Connect to test database
	db, err := sql.Open("pgx", "postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Ensure connectivity
	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available, skipping integration test: %v", err)
	}

	ctx := context.Background()

	// Clean up test data first
	_ = cleanupTestData(ctx, db)

	// Create queries
	queries := storegen.New(db)
	repo := store.New(queries)

	// Create a mock Stytch client that returns deterministic IDs
	mockStytch := &mockStytchClientComplete{
		orgID:   uuid.New().String(),
		memberID: uuid.New().String(),
	}

	svc := New(repo, mockStytch, db)

	// Setup test data: pending onboarding + ticket
	normalizedEmail := "test-provision-complete@example.com"

	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert pending onboarding: %v", err)
	}

	// Create onboarding ticket
	ticketToken, err := generateRandomString(32)
	if err != nil {
		t.Fatalf("generate ticket token: %v", err)
	}
	ticketHash := sha256Sum([]byte(ticketToken))
	encryptedIST := []byte("test-intermediate-session-token")

	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash,
		EncryptedIst:        encryptedIST,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create onboarding ticket: %v", err)
	}

	// Mock Stytch exchange
	mockStytch.exchangeSession = &stytch.DiscoveryExchangeResponse{
		SessionToken: "fake-stytch-session",
		ExpiresAt:    "",
	}

	// Execute provisioning
	idempotencyKey := "idem-" + uuid.New().String()
	data := ProvisionData{
		SchoolName:     "Kitengela Test School",
		IdempotencyKey: idempotencyKey,
		AdminName:      "Kitengela Test School",
	}

	result, err := svc.ProvisionTenant(ctx, ticketToken, data)
	if err != nil {
		t.Fatalf("ProvisionTenant failed: %v", err)
	}

	if result.TenantID == nil {
		t.Fatal("expected TenantID to be set")
	}
	if result.UserID == nil {
		t.Fatal("expected UserID to be set")
	}
	if result.SchoolID == nil {
		t.Fatal("expected SchoolID to be set")
	}

	// Verify tenant created
	tenant, err := repo.GetTenantByStytchOrgID(ctx, uuid.NullUUID{UUID: uuid.MustParse(mockStytch.orgID), Valid: true})
	if err != nil {
		t.Fatalf("tenant not found: %v", err)
	}
	if tenant.Name != "Kitengela Test School" {
		t.Errorf("tenant name = %q, want Kitengela Test School", tenant.Name)
	}

	// Verify pending onboarding updated
	pendingAfter, err := repo.GetPendingOnboardingByID(ctx, pending.ID)
	if err != nil {
		t.Fatalf("pending onboarding not found after provisioning: %v", err)
	}
	if pendingAfter.Status != "completed" {
		t.Errorf("pending status = %q, want completed", pendingAfter.Status)
	}
	if !pendingAfter.IdempotencyKey.Valid || pendingAfter.IdempotencyKey.String != idempotencyKey {
		t.Errorf("idempotency key not saved correctly")
	}

// 	// Test idempotency: second call with same key should not create duplicates
// 	result2, err := svc.ProvisionTenant(ctx, ticketToken, data)
// 	if err != nil {
// 		t.Fatalf("idempotency ProvisionTenant failed: %v", err)
// 	}
// 	if result2.TenantID == nil || *result2.TenantID != *result.TenantID {
// 		t.Errorf("idempotency failed: tenant IDs differ")
// 	}
// 
// 	// Cleanup
	_ = cleanupTestData(ctx, db)
}

// TestProvisionTenant_StytchFailure validates error handling when Stytch fails
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
	_ = cleanupTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	mockStytch := &mockStytchClientComplete{
		orgMemberErr: "stytch service unavailable",
	}

	svc := New(repo, mockStytch, db)

	email := "test-stytch-failure@example.com"
	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: email,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "testticket123"
	ticketHash := sha256Sum([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash,
		EncryptedIst:        []byte("ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	data := ProvisionData{
		SchoolName:     "Fail School",
		IdempotencyKey: "idem-fail",
		AdminName:      "Fail School",
	}

	_, err = svc.ProvisionTenant(ctx, ticketToken, data)
	if err == nil {
		t.Fatal("expected ProvisionTenant to fail when Stytch is down")
	}

	// Cleanup
	_ = cleanupTestData(ctx, db)
}

// mockStytchClientComplete implements stytch.Client for testing
type mockStytchClientComplete struct {
	orgID           string
	memberID        string
	orgMemberErr    string
	exchangeSession *stytch.DiscoveryExchangeResponse
	exchangeErr     string
}

func (m *mockStytchClientComplete) SendDiscoveryMagicLink(ctx context.Context, email, loginURL, pkceCodeChallenge string) error {
	return nil
}

func (m *mockStytchClientComplete) AuthenticateDiscovery(ctx context.Context, token, pkceCodeVerifier string) (*stytch.DiscoveryAuthResponse, error) {
	return &stytch.DiscoveryAuthResponse{
		Email:                    "test@example.com",
		IntermediateSessionToken: "ist",
		DiscoveredOrganizations:  nil,
	}, nil
}

func (m *mockStytchClientComplete) ExchangeDiscoverySession(ctx context.Context, intermediateSessionToken string) (*stytch.DiscoveryExchangeResponse, error) {
	if m.exchangeErr != "" {
		return nil, context.DeadlineExceeded
	}
	if m.exchangeSession != nil {
		return m.exchangeSession, nil
	}
	return &stytch.DiscoveryExchangeResponse{SessionToken: "token"}, nil
}

func (m *mockStytchClientComplete) CreateOrganizationAndMember(ctx context.Context, name, email string) (*stytch.OrgMemberResponse, error) {
	if m.orgMemberErr != "" {
		return nil, context.DeadlineExceeded
	}
	orgID := m.orgID
	memberID := m.memberID
	if orgID == "" {
		orgID = uuid.New().String()
	}
	if memberID == "" {
		memberID = uuid.New().String()
	}
	return &stytch.OrgMemberResponse{
		OrganizationID: orgID,
		MemberID:       memberID,
	}, nil
}

func (m *mockStytchClientComplete) RevokeMemberSession(ctx context.Context, memberID, sessionID string) error {
	return nil
}

// sha256Sum returns sha256 hash of data
func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// Helper for cleanup
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
	_ = cleanupTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)
	mockStytch := &mockStytchClientComplete{orgID: uuid.New().String(), memberID: uuid.New().String()}
	svc := New(repo, mockStytch, db)

	email := "test-completed-pending@example.com"
	tenantID := uuid.New()

	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: email,
		Status:                  "completed",
		CurrentStep:             "complete",
		FormData:                []byte(`{}`),
		StytchOrganizationID:    uuid.NullUUID{UUID: uuid.New(), Valid: true},
		StytchMemberID:          uuid.NullUUID{UUID: uuid.New(), Valid: true},
		TenantID:                uuid.NullUUID{UUID: tenantID, Valid: true},
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "token-completed"
	ticketHash := sha256Sum([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash,
		EncryptedIst:        []byte("ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	_, err = svc.ProvisionTenant(ctx, ticketToken, ProvisionData{SchoolName: "Blocked", IdempotencyKey: "idem-1", AdminName: "Blocked"})
	if err == nil {
		t.Fatal("expected error when pending status is completed")
	}

	_ = cleanupTestData(ctx, db)
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
	_ = cleanupTestData(ctx, db)

	queries := storegen.New(db)
	repo := store.New(queries)

	email := "test-existing-user@example.com"

	// Create a tenant and existing user globally for this email
	existingName := "Existing " + uuid.New().String()[:8]
	tenant, err := repo.CreateTenant(ctx, storegen.CreateTenantParams{Name: existingName, StytchOrganizationID: uuid.NullUUID{UUID: uuid.New(), Valid: true}})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	_, err = repo.CreateUser(ctx, storegen.CreateUserParams{
		TenantID:        tenant.ID,
		EmailNormalized: email,
		StytchMemberID:  uuid.NullUUID{UUID: uuid.New(), Valid: true},
		Name:            "Existing User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	mockStytch := &mockStytchClientComplete{orgID: uuid.New().String(), memberID: uuid.New().String()}
	svc := New(repo, mockStytch, db)

	pending, err := repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: email,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert pending: %v", err)
	}

	ticketToken := "token-existing-user"
	ticketHash := sha256Sum([]byte(ticketToken))
	_, err = repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash,
		EncryptedIst:        []byte("ist"),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	_, err = svc.ProvisionTenant(ctx, ticketToken, ProvisionData{SchoolName: "Blocked", IdempotencyKey: "idem-2", AdminName: "Blocked"})
	if err == nil {
		t.Fatal("expected error when user already exists for email")
	}

	_ = cleanupTestData(ctx, db)
}

func cleanupTestData(ctx context.Context, db *sql.DB) error {
	// Delete test emails
	_, _ = db.ExecContext(ctx, `DELETE FROM membership WHERE user_id IN (SELECT id FROM "user" WHERE email_normalized LIKE 'test-%')`)
	_, _ = db.ExecContext(ctx, `DELETE FROM school WHERE tenant_id IN (SELECT id FROM tenant WHERE name LIKE 'Kitengela Test%' OR name LIKE 'Fail%')`)
	_, _ = db.ExecContext(ctx, `DELETE FROM "user" WHERE email_normalized LIKE 'test-%'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM tenant WHERE name LIKE 'Kitengela Test%' OR name LIKE 'Fail%'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM onboarding_ticket WHERE pending_onboarding_id IN (SELECT id FROM pending_onboarding WHERE verified_email_normalized LIKE 'test-%')`)
	_, _ = db.ExecContext(ctx, `DELETE FROM pending_onboarding WHERE verified_email_normalized LIKE 'test-%'`)
	return nil
}
