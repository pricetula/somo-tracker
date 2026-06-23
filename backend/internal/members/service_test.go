package members

import (
	"context"
	"fmt"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listByRoleFn        func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	getActiveSchoolIDFn func(ctx context.Context, tenantID, userID string) (string, error)
	listInvitationsFn   func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error)
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

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc  *Service
	repo *MockRepository
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}

	svc := &Service{
		repo: repo,
	}

	return &testHarness{
		svc:  svc,
		repo: repo,
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
