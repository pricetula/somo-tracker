package invitations

import (
	"context"
	"fmt"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

// MockSchoolResolver satisfies SchoolResolver for handler tests.
type MockSchoolResolver struct {
	getActiveSchoolIDFn func(ctx context.Context, tenantID, userID string) (string, error)
}

func (m *MockSchoolResolver) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if m.getActiveSchoolIDFn != nil {
		return m.getActiveSchoolIDFn(ctx, tenantID, userID)
	}
	return "school_001", nil
}

type MockRepository struct {
	listInvitationsFn func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error)
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
	svc := &Service{repo: repo}
	return &testHarness{svc: svc, repo: repo}
}

// ============================================================================
// Tests: ListInvitations
// ============================================================================

func TestListInvitations_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		fullName := "Alice"
		return []Invitation{
			{ID: "inv_001", Email: "alice@school.com", Role: "TEACHER", Status: "pending", FullName: &fullName},
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
		fullName := "Alice"
		return []Invitation{
			{ID: "inv_001", Email: "alice@school.com", Role: "TEACHER", Status: "pending", FullName: &fullName},
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

func TestListInvitations_InvalidLimit(t *testing.T) {
	h := newTestHarness()

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		if filter.Limit != 50 {
			return nil, 0, fmt.Errorf("expected limit 50, got %d", filter.Limit)
		}
		return []Invitation{}, 0, nil
	}

	_, _, err := h.svc.ListInvitations(context.Background(), "tenant_001", "school_001", ListInvitationsFilter{
		Limit: 0, Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListInvitations_NegativeOffset(t *testing.T) {
	h := newTestHarness()

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		if filter.Offset != 0 {
			return nil, 0, fmt.Errorf("expected offset 0, got %d", filter.Offset)
		}
		return []Invitation{}, 0, nil
	}

	_, _, err := h.svc.ListInvitations(context.Background(), "tenant_001", "school_001", ListInvitationsFilter{
		Limit: 50, Offset: -5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
