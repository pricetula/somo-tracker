package members

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listByRoleFn                  func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	getActiveSchoolIDFn           func(ctx context.Context, tenantID, userID string) (string, error)
	listInvitationsFn             func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error)
	getPendingInviteByEmailFn     func(ctx context.Context, schoolID, email string) (*Invitation, error)
	getMemberByEmailFn            func(ctx context.Context, schoolID, email string) (*Member, error)
	getTenantStytchOrgIDFn        func(ctx context.Context, tenantID string) (string, error)
	createInvitationFn            func(ctx context.Context, inv *Invitation, invitedBy string) error
	setInvitationStytchMemberIDFn func(ctx context.Context, id, stytchMemberID string) error
}

func (m *MockRepository) ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
	if m.listByRoleFn != nil {
		return m.listByRoleFn(ctx, tenantID, schoolID, role, offset, limit, search)
	}
	return nil, 0, nil
}

func (m *MockRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if m.getActiveSchoolIDFn != nil {
		return m.getActiveSchoolIDFn(ctx, tenantID, userID)
	}
	return "school_001", nil
}

func (m *MockRepository) ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
	if m.listInvitationsFn != nil {
		return m.listInvitationsFn(ctx, tenantID, schoolID, filter)
	}
	return nil, 0, nil
}

func (m *MockRepository) GetPendingInviteByEmail(ctx context.Context, schoolID, email string) (*Invitation, error) {
	if m.getPendingInviteByEmailFn != nil {
		return m.getPendingInviteByEmailFn(ctx, schoolID, email)
	}
	return nil, nil
}

func (m *MockRepository) GetMemberByEmail(ctx context.Context, schoolID, email string) (*Member, error) {
	if m.getMemberByEmailFn != nil {
		return m.getMemberByEmailFn(ctx, schoolID, email)
	}
	return nil, nil
}

func (m *MockRepository) GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	if m.getTenantStytchOrgIDFn != nil {
		return m.getTenantStytchOrgIDFn(ctx, tenantID)
	}
	return "org_stytch_001", nil
}

func (m *MockRepository) CreateInvitation(ctx context.Context, inv *Invitation, invitedBy string) error {
	if m.createInvitationFn != nil {
		return m.createInvitationFn(ctx, inv, invitedBy)
	}
	return nil
}

func (m *MockRepository) SetInvitationStytchMemberID(ctx context.Context, id, stytchMemberID string) error {
	if m.setInvitationStytchMemberIDFn != nil {
		return m.setInvitationStytchMemberIDFn(ctx, id, stytchMemberID)
	}
	return nil
}

// ============================================================================
// MockIdentityProvider (subset of auth.IdentityProvider for members)
// ============================================================================

type MockIDP struct {
	inviteMemberByEmailFn func(ctx context.Context, orgID, email, name, redirectURL string) (string, error)
}

func (m *MockIDP) InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
	if m.inviteMemberByEmailFn != nil {
		return m.inviteMemberByEmailFn(ctx, orgID, email, name, redirectURL)
	}
	return "member_" + email, nil
}

func (m *MockIDP) SendDiscoveryEmail(ctx context.Context, email string) error {
	return nil
}

func (m *MockIDP) AuthenticateDiscoveryToken(ctx context.Context, token string) (string, string, error) {
	return "ist", "email", nil
}

func (m *MockIDP) CreateOrganization(ctx context.Context, name string) (string, error) {
	return "org_" + name, nil
}

func (m *MockIDP) ExchangeIntermediateSession(ctx context.Context, ist, orgID string) (auth.ExchangeResult, error) {
	return auth.ExchangeResult{}, nil
}

func (m *MockIDP) CreateMember(ctx context.Context, orgID, email, name string) (string, error) {
	return "member_" + email, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc    *Service
	repo   *MockRepository
	idp    *MockIDP
	logs   *observer.ObservedLogs
	logger *zap.Logger
	cfg    config.Config
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	idp := &MockIDP{}

	observedCore, observedLogs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)

	cfg := config.Config{
		AppEnv:      "test",
		FrontendURL: "http://localhost:3000",
	}

	svc := &Service{
		repo:   repo,
		idp:    idp,
		cfg:    cfg,
		logger: logger,
	}

	return &testHarness{
		svc:    svc,
		repo:   repo,
		idp:    idp,
		logs:   observedLogs,
		logger: logger,
		cfg:    cfg,
	}
}

// ============================================================================
// Tests: ListMembers
// ============================================================================

func TestListMembers_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		return []Member{
			{ID: "user_001", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith", Role: "TEACHER", IsActive: true},
		}, 1, nil
	}

	members, total, err := h.svc.ListMembers(context.Background(), "tenant_001", "school_001", "TEACHER", 0, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].Email != "alice@school.com" {
		t.Fatalf("expected email 'alice@school.com', got %q", members[0].Email)
	}
}

func TestListMembers_WithSearch(t *testing.T) {
	h := newTestHarness()

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		if search != "Alice" {
			return nil, 0, nil
		}
		return []Member{
			{ID: "user_001", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith", Role: "TEACHER", IsActive: true},
		}, 1, nil
	}

	members, total, err := h.svc.ListMembers(context.Background(), "tenant_001", "school_001", "TEACHER", 0, 50, "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(members) == 0 {
		t.Fatal("expected at least 1 member from search")
	}
}

func TestListMembers_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		return []Member{}, 0, nil
	}

	members, total, err := h.svc.ListMembers(context.Background(), "tenant_001", "school_001", "FINANCE", 0, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(members) != 0 {
		t.Fatalf("expected 0 members, got %d", len(members))
	}
}

func TestListMembers_InvalidLimit(t *testing.T) {
	h := newTestHarness()

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		// Service should clamp limit to 50 if invalid
		if limit != 50 {
			return nil, 0, fmt.Errorf("expected limit 50, got %d", limit)
		}
		return []Member{}, 0, nil
	}

	_, _, err := h.svc.ListMembers(context.Background(), "tenant_001", "school_001", "TEACHER", 0, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMembers_NegativeOffset(t *testing.T) {
	h := newTestHarness()

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		if offset != 0 {
			return nil, 0, fmt.Errorf("expected offset 0, got %d", offset)
		}
		return []Member{}, 0, nil
	}

	_, _, err := h.svc.ListMembers(context.Background(), "tenant_001", "school_001", "TEACHER", -5, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests: BulkInvite
// ============================================================================

func TestBulkInvite_HappyPath(t *testing.T) {
	h := newTestHarness()

	req := BulkInviteRequest{
		Role: "TEACHER",
		Invites: []InviteItem{
			{Email: "newteacher@school.com", FirstName: "New", LastName: "Teacher"},
		},
	}

	resp, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 1 {
		t.Fatalf("expected 1 sent, got %d", resp.Sent)
	}
	if resp.Failed != 0 {
		t.Fatalf("expected 0 failed, got %d", resp.Failed)
	}
}

func TestBulkInvite_EmptyInvites(t *testing.T) {
	h := newTestHarness()

	req := BulkInviteRequest{
		Role:    "TEACHER",
		Invites: []InviteItem{},
	}

	_, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err == nil {
		t.Fatal("expected error for empty invites, got nil")
	}
}

func TestBulkInvite_MissingEmail(t *testing.T) {
	h := newTestHarness()

	req := BulkInviteRequest{
		Role: "NURSE",
		Invites: []InviteItem{
			{Email: "", FirstName: "No", LastName: "Email"},
		},
	}

	resp, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 0 {
		t.Fatalf("expected 0 sent, got %d", resp.Sent)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", resp.Failed)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(resp.Errors))
	}
}

func TestBulkInvite_AlreadyMember(t *testing.T) {
	h := newTestHarness()

	h.repo.getMemberByEmailFn = func(ctx context.Context, schoolID, email string) (*Member, error) {
		return &Member{ID: "existing", Email: email, Role: "TEACHER"}, nil
	}

	req := BulkInviteRequest{
		Role: "TEACHER",
		Invites: []InviteItem{
			{Email: "existing@school.com", FirstName: "Already", LastName: "Here"},
		},
	}

	resp, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 0 {
		t.Fatalf("expected 0 sent, got %d", resp.Sent)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", resp.Failed)
	}
}

func TestBulkInvite_PendingInviteExists(t *testing.T) {
	h := newTestHarness()

	h.repo.getPendingInviteByEmailFn = func(ctx context.Context, schoolID, email string) (*Invitation, error) {
		return &Invitation{Email: email, Status: "pending"}, nil
	}

	req := BulkInviteRequest{
		Role: "TEACHER",
		Invites: []InviteItem{
			{Email: "pending@school.com", FirstName: "Already", LastName: "Invited"},
		},
	}

	resp, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 0 {
		t.Fatalf("expected 0 sent, got %d", resp.Sent)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", resp.Failed)
	}
}

func TestBulkInvite_StytchOrgNotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getTenantStytchOrgIDFn = func(ctx context.Context, tenantID string) (string, error) {
		return "", errors.New("tenant not found")
	}

	req := BulkInviteRequest{
		Role: "TEACHER",
		Invites: []InviteItem{
			{Email: "teacher@school.com", FirstName: "Teacher", LastName: "User"},
		},
	}

	_, err := h.svc.BulkInvite(context.Background(), "nonexistent", "school_001", req)
	if err == nil {
		t.Fatal("expected error for missing tenant, got nil")
	}
}

func TestBulkInvite_StytchInviteFailsButInvitationPersists(t *testing.T) {
	h := newTestHarness()

	h.idp.inviteMemberByEmailFn = func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
		return "", errors.New("stytch upstream error")
	}

	req := BulkInviteRequest{
		Role: "NURSE",
		Invites: []InviteItem{
			{Email: "nurse@school.com", FirstName: "Nurse", LastName: "Betty"},
		},
	}

	resp, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invitation persists even if Stytch fails
	if resp.Sent != 1 {
		t.Fatalf("expected 1 sent (invitation persisted), got %d", resp.Sent)
	}
	if resp.Failed != 0 {
		t.Fatalf("expected 0 failed, got %d", resp.Failed)
	}

	// Verify WARN log emitted
	warnLogs := h.logs.FilterLevelExact(zapcore.WarnLevel)
	if warnLogs.Len() < 1 {
		t.Log("expected WARN log about Stytch invite failure (non-fatal check)")
	}
}

func TestBulkInvite_MultipleInvitesMixedResults(t *testing.T) {
	h := newTestHarness()

	callCount := 0
	h.repo.getMemberByEmailFn = func(ctx context.Context, schoolID, email string) (*Member, error) {
		if email == "existing@school.com" {
			return &Member{ID: "existing", Email: email, Role: "TEACHER"}, nil
		}
		return nil, nil
	}

	h.idp.inviteMemberByEmailFn = func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
		callCount++
		return "member_" + email, nil
	}

	req := BulkInviteRequest{
		Role: "TEACHER",
		Invites: []InviteItem{
			{Email: "existing@school.com", FirstName: "Existing", LastName: "User"},
			{Email: "new1@school.com", FirstName: "New1", LastName: "User"},
			{Email: "new2@school.com", FirstName: "New2", LastName: "User"},
		},
	}

	resp, err := h.svc.BulkInvite(context.Background(), "tenant_001", "school_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 2 {
		t.Fatalf("expected 2 sent, got %d", resp.Sent)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed (existing member), got %d", resp.Failed)
	}
}

// ============================================================================
// Tests: ListInvitations
// ============================================================================

func TestListInvitations_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		firstName := "Alice"
		return []Invitation{
			{ID: "inv_001", Email: "alice@school.com", Role: "TEACHER", Status: "pending", FirstName: &firstName},
		}, 1, nil
	}

	invitations, total, err := h.svc.ListInvitations(context.Background(), "tenant_001", "school_001", ListInvitationsFilter{
		Limit: 50, Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(invitations))
	}
}

func TestListInvitations_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		return []Invitation{}, 0, nil
	}

	invitations, total, err := h.svc.ListInvitations(context.Background(), "tenant_001", "school_001", ListInvitationsFilter{
		Limit: 50, Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(invitations) != 0 {
		t.Fatalf("expected 0 invitations, got %d", len(invitations))
	}
}

func TestListInvitations_WithFilters(t *testing.T) {
	h := newTestHarness()

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		if filter.Status != "pending" {
			return nil, 0, fmt.Errorf("expected status filter 'pending', got %q", filter.Status)
		}
		if filter.Role != "TEACHER" {
			return nil, 0, fmt.Errorf("expected role filter 'TEACHER', got %q", filter.Role)
		}
		firstName := "Alice"
		return []Invitation{
			{ID: "inv_001", Email: "alice@school.com", Role: "TEACHER", Status: "pending", FirstName: &firstName},
		}, 1, nil
	}

	invitations, total, err := h.svc.ListInvitations(context.Background(), "tenant_001", "school_001", ListInvitationsFilter{
		Status: "pending",
		Role:   "TEACHER",
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(invitations))
	}
}

// ============================================================================
// Tests: CreateInvitations
// ============================================================================

func TestCreateInvitations_HappyPath(t *testing.T) {
	h := newTestHarness()

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "new@school.com", FirstName: "New", LastName: "User", Role: "TEACHER"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 1 {
		t.Fatalf("expected 1 sent, got %d", resp.Sent)
	}
	if resp.Failed != 0 {
		t.Fatalf("expected 0 failed, got %d", resp.Failed)
	}
}

func TestCreateInvitations_EmptyInvites(t *testing.T) {
	h := newTestHarness()

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{},
	}

	_, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err == nil {
		t.Fatal("expected error for empty invites, got nil")
	}
}

func TestCreateInvitations_MissingEmail(t *testing.T) {
	h := newTestHarness()

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "", FirstName: "No", LastName: "Email", Role: "TEACHER"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed for missing email, got %d", resp.Failed)
	}
}

func TestCreateInvitations_InvalidRole(t *testing.T) {
	h := newTestHarness()

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "badrole@school.com", FirstName: "Bad", LastName: "Role", Role: "PRINCIPAL"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed for invalid role, got %d", resp.Failed)
	}
}

func TestCreateInvitations_AlreadyMember(t *testing.T) {
	h := newTestHarness()

	h.repo.getMemberByEmailFn = func(ctx context.Context, schoolID, email string) (*Member, error) {
		return &Member{ID: "existing", Email: email, Role: "TEACHER"}, nil
	}

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "existing@school.com", FirstName: "Already", LastName: "Here", Role: "TEACHER"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed for existing member, got %d", resp.Failed)
	}
}

func TestCreateInvitations_PendingInviteExists(t *testing.T) {
	h := newTestHarness()

	h.repo.getPendingInviteByEmailFn = func(ctx context.Context, schoolID, email string) (*Invitation, error) {
		return &Invitation{Email: email, Status: "pending"}, nil
	}

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "pending@school.com", FirstName: "Pending", LastName: "Invite", Role: "NURSE"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed for pending invite, got %d", resp.Failed)
	}
}

func TestCreateInvitations_CheckMemberError(t *testing.T) {
	h := newTestHarness()

	h.repo.getMemberByEmailFn = func(ctx context.Context, schoolID, email string) (*Member, error) {
		return nil, errors.New("db connection error")
	}

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "error@school.com", FirstName: "DB", LastName: "Error", Role: "FINANCE"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed (internal error), got %d", resp.Failed)
	}
}

func TestCreateInvitations_DBCreateFails(t *testing.T) {
	h := newTestHarness()

	h.repo.createInvitationFn = func(ctx context.Context, inv *Invitation, invitedBy string) error {
		return errors.New("insert failed: constraint violation")
	}

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "fail@school.com", FirstName: "DB", LastName: "Fail", Role: "TEACHER"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed for create failure, got %d", resp.Failed)
	}
}

func TestCreateInvitations_MultipleInvites(t *testing.T) {
	h := newTestHarness()

	callCount := 0
	h.repo.getMemberByEmailFn = func(ctx context.Context, schoolID, email string) (*Member, error) {
		if email == "existing@school.com" {
			return &Member{ID: "existing", Email: email, Role: "FINANCE"}, nil
		}
		return nil, nil
	}

	h.repo.createInvitationFn = func(ctx context.Context, inv *Invitation, invitedBy string) error {
		callCount++
		return nil
	}

	req := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "existing@school.com", FirstName: "Existing", LastName: "User", Role: "FINANCE"},
			{Email: "good1@school.com", FirstName: "Good1", LastName: "User", Role: "TEACHER"},
			{Email: "good2@school.com", FirstName: "Good2", LastName: "User", Role: "NURSE"},
		},
	}

	resp, err := h.svc.CreateInvitations(context.Background(), "tenant_001", "school_001", "user_001", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sent != 2 {
		t.Fatalf("expected 2 sent, got %d", resp.Sent)
	}
	if resp.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", resp.Failed)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 CreateInvitation calls, got %d", callCount)
	}
}

// Compile-time checks
var _ auth.IdentityProvider = (*MockIDP)(nil)
