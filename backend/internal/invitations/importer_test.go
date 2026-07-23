package invitations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// Mocks
// ============================================================================

// mockStytch implements StytchInviteSender for testing.
type mockStytch struct {
	createMemberFn         func(ctx context.Context, orgID, email, name string) (string, error)
	sendDiscoveryFn        func(ctx context.Context, email string) error
	sendDiscoveryWithURLFn func(ctx context.Context, email, redirectURL string) error
	getMemberFn            func(ctx context.Context, orgID, email string) (string, error)
	inviteMemberByEmailFn  func(ctx context.Context, orgID, email, name, redirectURL string) (string, error)
}

func (m *mockStytch) CreateMember(ctx context.Context, orgID, email, name string) (string, error) {
	if m.createMemberFn != nil {
		return m.createMemberFn(ctx, orgID, email, name)
	}
	return "member_id", nil
}

func (m *mockStytch) SendDiscoveryEmail(ctx context.Context, email string) error {
	if m.sendDiscoveryFn != nil {
		return m.sendDiscoveryFn(ctx, email)
	}
	return nil
}

func (m *mockStytch) SendDiscoveryEmailWithRedirect(ctx context.Context, email, redirectURL string) error {
	if m.sendDiscoveryWithURLFn != nil {
		return m.sendDiscoveryWithURLFn(ctx, email, redirectURL)
	}
	// Fall back to sendDiscoveryFn for backward compatibility
	if m.sendDiscoveryFn != nil {
		return m.sendDiscoveryFn(ctx, email)
	}
	return nil
}

func (m *mockStytch) GetMemberByEmail(ctx context.Context, orgID, email string) (string, error) {
	if m.getMemberFn != nil {
		return m.getMemberFn(ctx, orgID, email)
	}
	return "member_id", nil
}

func (m *mockStytch) InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
	if m.inviteMemberByEmailFn != nil {
		return m.inviteMemberByEmailFn(ctx, orgID, email, name, redirectURL)
	}
	return "stytch_member_abc123", nil
}

// mockImportTx implements pgx.Tx minimally for testing savepoints.
type mockImportTx struct {
	pgx.Tx
	execFn func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (m *mockImportTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

// mockRepo implements the Repository subset needed by StaffInviteImporter.
type mockRepo struct {
	insertInvitationFn func(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error
	checkEmailsFn      func(ctx context.Context, tenantID, schoolID string, emails []string) ([]string, []string, error)
	getStytchOrgIDFn   func(ctx context.Context, tenantID string) (string, error)
	listInvitationsFn  func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error)
	countInvitationsFn func(ctx context.Context, tenantID, schoolID string, role string) (int, error)
	revokeInvitationFn func(ctx context.Context, id, schoolID string) error
}

func (m *mockRepo) InsertInvitation(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error {
	if m.insertInvitationFn != nil {
		return m.insertInvitationFn(ctx, tx, params)
	}
	return nil
}

func (m *mockRepo) CheckExistingEmails(ctx context.Context, tenantID, schoolID string, emails []string) ([]string, []string, error) {
	if m.checkEmailsFn != nil {
		return m.checkEmailsFn(ctx, tenantID, schoolID, emails)
	}
	return nil, nil, nil
}

func (m *mockRepo) GetStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	if m.getStytchOrgIDFn != nil {
		return m.getStytchOrgIDFn(ctx, tenantID)
	}
	return "org_test_123", nil
}

func (m *mockRepo) ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
	if m.listInvitationsFn != nil {
		return m.listInvitationsFn(ctx, tenantID, schoolID, filter)
	}
	return nil, 0, nil
}

func (m *mockRepo) RevokeInvitation(ctx context.Context, id, schoolID string) error {
	if m.revokeInvitationFn != nil {
		return m.revokeInvitationFn(ctx, id, schoolID)
	}
	return nil
}

func (m *mockRepo) CountInvitations(ctx context.Context, tenantID, schoolID string, role string) (int, error) {
	if m.countInvitationsFn != nil {
		return m.countInvitationsFn(ctx, tenantID, schoolID, role)
	}
	return 0, nil
}

// ============================================================================
// Test: StaffInviteImporter.InsertOne — Happy Path
// ============================================================================

func TestStaffInviteImporter_InsertOne_HappyPath(t *testing.T) {
	var capturedOrgID, capturedEmail, capturedName, capturedRedirectURL string

	stytch := &mockStytch{
		inviteMemberByEmailFn: func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
			capturedOrgID = orgID
			capturedEmail = email
			capturedName = name
			capturedRedirectURL = redirectURL
			return "stytch_member_abc123", nil
		},
	}

	var insertedInvitation *InsertInvitationParams
	repo := &mockRepo{
		insertInvitationFn: func(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error {
			insertedInvitation = &params
			return nil
		},
	}

	imp := NewStaffInviteImporter(repo)
	imp.SetStytchAdapter(stytch)
	imp.SetBackendURL("http://localhost:3030")

	aug := augmentedInviteRow{
		Email:       "newteacher@school.com",
		FullName:    "Jane Teacher",
		TenantID:    "11111111-1111-4111-8111-111111111111",
		SchoolID:    "22222222-2222-4222-8222-222222222222",
		Role:        "TEACHER",
		StytchOrgID: "org_test_123",
		InvitedBy:   "user_001",
		ImportJobID: "job_001",
	}
	raw, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: raw, StagingRowID: uuid.New()}

	err := imp.InsertOne(context.Background(), &mockImportTx{}, row)
	if err != nil {
		t.Fatalf("InsertOne failed: %v", err)
	}

	if insertedInvitation == nil {
		t.Fatal("expected InsertInvitation to be called")
	}
	if insertedInvitation.Email != "newteacher@school.com" {
		t.Errorf("expected email newteacher@school.com, got %s", insertedInvitation.Email)
	}
	if insertedInvitation.Role != "TEACHER" {
		t.Errorf("expected role TEACHER, got %s", insertedInvitation.Role)
	}
	if insertedInvitation.Status != "pending" {
		t.Errorf("expected status pending, got %s", insertedInvitation.Status)
	}
	if insertedInvitation.StytchMemberID != "stytch_member_abc123" {
		t.Errorf("expected StytchMemberID 'stytch_member_abc123', got %s", insertedInvitation.StytchMemberID)
	}

	// Verify InviteMemberByEmail was called with correct params
	if capturedEmail != "newteacher@school.com" {
		t.Errorf("expected email 'newteacher@school.com' passed to InviteMemberByEmail, got %q", capturedEmail)
	}
	if capturedOrgID != "org_test_123" {
		t.Errorf("expected orgID 'org_test_123', got %q", capturedOrgID)
	}
	if capturedName != "Jane Teacher" {
		t.Errorf("expected name 'Jane Teacher', got %q", capturedName)
	}
	if capturedRedirectURL != "http://localhost:3030"+StaffInviteRedirectPath {
		t.Errorf("expected redirect URL ending with %q, got %q", StaffInviteRedirectPath, capturedRedirectURL)
	}
}

// ============================================================================
// Test: InsertOne — Stytch invite email API error sanitized
// ============================================================================

func TestStaffInviteImporter_InsertOne_StytchErrorSanitized(t *testing.T) {
	// InviteMemberByEmail fails with a verbose Stytch error
	stytch := &mockStytch{
		inviteMemberByEmailFn: func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
			return "", fmt.Errorf("internal_error: stytch invite: A Stytch API error occurred")
		},
	}

	repo := &mockRepo{}
	imp := NewStaffInviteImporter(repo)
	imp.SetStytchAdapter(stytch)
	imp.SetBackendURL("http://localhost:3030")

	aug := augmentedInviteRow{
		Email:       "badredirect@school.com",
		FullName:    "Bad Redirect",
		TenantID:    "11111111-1111-4111-8111-111111111111",
		SchoolID:    "22222222-2222-4222-8222-222222222222",
		Role:        "TEACHER",
		StytchOrgID: "org_test_123",
		InvitedBy:   "user_001",
		ImportJobID: "job_001",
	}
	raw, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: raw, StagingRowID: uuid.New()}

	err := imp.InsertOne(context.Background(), &mockImportTx{}, row)
	if err == nil {
		t.Fatal("expected error from Stytch invite failure")
	}

	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureStytchAPIError {
		t.Errorf("expected ImportFailureStytchAPIError, got %s", impErr.Type)
	}

	// The error message should NOT contain raw Stytch internals like request ID or status code
	msg := impErr.Message
	if containsStytchRaw(msg) {
		t.Errorf("error message should not contain raw Stytch internals (request ID, status code, error_url), got: %s", msg)
	}
}

// containsStytchRaw checks for strings that appear in raw Stytch errors.
func containsStytchRaw(msg string) bool {
	redFlags := []string{
		"Stytch Error",
		"request ID:",
		"status code:",
		"error_url:",
		"error_details:",
	}
	for _, flag := range redFlags {
		if contains(msg, flag) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Test: InsertOne — Stytch invite email error (non-duplicate)
// ============================================================================

func TestStaffInviteImporter_InsertOne_InviteEmailError(t *testing.T) {
	stytch := &mockStytch{
		inviteMemberByEmailFn: func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
			return "", fmt.Errorf("internal_error: stytch invite: Some random Stytch failure")
		},
	}

	repo := &mockRepo{}
	imp := NewStaffInviteImporter(repo)
	imp.SetStytchAdapter(stytch)
	imp.SetBackendURL("http://localhost:3030")

	aug := augmentedInviteRow{
		Email:       "failinvite@school.com",
		FullName:    "Fail Invite",
		TenantID:    "11111111-1111-4111-8111-111111111111",
		SchoolID:    "22222222-2222-4222-8222-222222222222",
		Role:        "TEACHER",
		StytchOrgID: "org_test_123",
		InvitedBy:   "user_001",
		ImportJobID: "job_001",
	}
	raw, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: raw, StagingRowID: uuid.New()}

	err := imp.InsertOne(context.Background(), &mockImportTx{}, row)
	if err == nil {
		t.Fatal("expected error from InviteMemberByEmail failure")
	}

	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureStytchAPIError {
		t.Errorf("expected ImportFailureStytchAPIError, got %s", impErr.Type)
	}
	if containsStytchRaw(impErr.Message) {
		t.Errorf("error message should not contain raw Stytch internals, got: %s", impErr.Message)
	}
}

// ============================================================================
// Test: InsertOne — DB unique constraint violation on invitation insert
// ============================================================================

func TestStaffInviteImporter_InsertOne_DuplicateEmailDB(t *testing.T) {
	stytch := &mockStytch{}
	repo := &mockRepo{
		insertInvitationFn: func(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error {
			// Simulate PostgreSQL unique constraint violation
			return &pgconn.PgError{Code: "23505"}
		},
	}

	imp := NewStaffInviteImporter(repo)
	imp.SetStytchAdapter(stytch)
	imp.SetBackendURL("http://localhost:3030")

	aug := augmentedInviteRow{
		Email:       "duplicate@school.com",
		FullName:    "Dup User",
		TenantID:    "11111111-1111-4111-8111-111111111111",
		SchoolID:    "22222222-2222-4222-8222-222222222222",
		Role:        "TEACHER",
		StytchOrgID: "org_test_123",
		InvitedBy:   "user_001",
		ImportJobID: "job_001",
	}
	raw, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: raw, StagingRowID: uuid.New()}

	err := imp.InsertOne(context.Background(), &mockImportTx{}, row)
	if err == nil {
		t.Fatal("expected error from duplicate email")
	}

	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureDuplicateEmail {
		t.Errorf("expected ImportFailureDuplicateEmail, got %s", impErr.Type)
	}
}

// ============================================================================
// Test: InsertOne — Stytch not configured
// ============================================================================

func TestStaffInviteImporter_InsertOne_StytchNotConfigured(t *testing.T) {
	repo := &mockRepo{}
	imp := NewStaffInviteImporter(repo)
	// Intentionally NOT setting stytch adapter

	aug := augmentedInviteRow{
		Email:       "nostytch@school.com",
		FullName:    "No Stytch",
		TenantID:    "11111111-1111-4111-8111-111111111111",
		SchoolID:    "22222222-2222-4222-8222-222222222222",
		Role:        "TEACHER",
		StytchOrgID: "org_test_123",
		InvitedBy:   "user_001",
		ImportJobID: "job_001",
	}
	raw, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: raw, StagingRowID: uuid.New()}

	err := imp.InsertOne(context.Background(), &mockImportTx{}, row)
	if err == nil {
		t.Fatal("expected error when stytch not configured")
	}

	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureStytchAPIError {
		t.Errorf("expected ImportFailureStytchAPIError, got %s", impErr.Type)
	}
}

// ============================================================================
// Test: InsertOne — Stytch invite email sends to correct email and org
// ============================================================================

func TestStaffInviteImporter_InsertOne_SendsToCorrectEmail(t *testing.T) {
	var capturedOrgID, capturedEmail string

	stytch := &mockStytch{
		inviteMemberByEmailFn: func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
			capturedOrgID = orgID
			capturedEmail = email
			return "stytch_member_abc123", nil
		},
	}

	repo := &mockRepo{}
	imp := NewStaffInviteImporter(repo)
	imp.SetStytchAdapter(stytch)
	imp.SetBackendURL("http://localhost:3030")

	aug := augmentedInviteRow{
		Email:       "alice@school.com",
		FullName:    "Alice Johnson",
		TenantID:    "11111111-1111-4111-8111-111111111111",
		SchoolID:    "22222222-2222-4222-8222-222222222222",
		Role:        "NURSE",
		StytchOrgID: "org_test_123",
		InvitedBy:   "user_001",
		ImportJobID: "job_001",
	}
	raw, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: raw, StagingRowID: uuid.New()}

	err := imp.InsertOne(context.Background(), &mockImportTx{}, row)
	if err != nil {
		t.Fatalf("InsertOne failed: %v", err)
	}

	if capturedEmail != "alice@school.com" {
		t.Errorf("expected email 'alice@school.com' passed to InviteMemberByEmail, got %q", capturedEmail)
	}
	if capturedOrgID != "org_test_123" {
		t.Errorf("expected orgID 'org_test_123' passed to InviteMemberByEmail, got %q", capturedOrgID)
	}
}
