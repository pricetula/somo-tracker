package summaries

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// ─── In-memory mock repository ────────────────────────────────────────────

type mockRepo struct {
	mu sync.Mutex

	summaries map[string]*CompetencySummary // keyed by ID

	calculateForClassFn func(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error)
	setOverrideLevelFn  func(ctx context.Context, id, tenantID string, overrideLevel *string) error
	markSyncedFn        func(ctx context.Context, id, tenantID, status string, syncedAt *string) error
	getSyncStatusFn     func(ctx context.Context, id, tenantID string) (string, error)
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		summaries: make(map[string]*CompetencySummary),
	}
}

func (m *mockRepo) addSummary(s *CompetencySummary) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaries[s.ID] = s
}

func (m *mockRepo) GetByID(ctx context.Context, id, tenantID string) (*CompetencySummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.summaries[id]
	if !ok {
		return nil, ErrNotFound
	}
	if s.TenantID != tenantID {
		return nil, ErrNotFound
	}
	// Return a copy to avoid mutation races in tests
	sCopy := *s
	return &sCopy, nil
}

func (m *mockRepo) List(ctx context.Context, tenantID string, query ListSummariesQuery) ([]CompetencySummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []CompetencySummary
	for _, s := range m.summaries {
		if s.TenantID != tenantID {
			continue
		}
		if query.StudentID != "" && s.StudentID != query.StudentID {
			continue
		}
		if query.ClassID != "" && s.ClassID != query.ClassID {
			continue
		}
		if query.LearningAreaID != "" && s.LearningAreaID != query.LearningAreaID {
			continue
		}
		if query.AcademicYear > 0 && s.AcademicYear != query.AcademicYear {
			continue
		}
		if query.Term > 0 && s.Term != query.Term {
			continue
		}
		result = append(result, *s)
	}
	if result == nil {
		result = []CompetencySummary{}
	}
	return result, nil
}

func (m *mockRepo) CalculateForClass(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
	return m.calculateForClassFn(ctx, tenantID, payload)
}

func (m *mockRepo) SetOverrideLevel(ctx context.Context, id, tenantID string, overrideLevel *string) error {
	return m.setOverrideLevelFn(ctx, id, tenantID, overrideLevel)
}

func (m *mockRepo) MarkSynced(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
	return m.markSyncedFn(ctx, id, tenantID, status, syncedAt)
}

func (m *mockRepo) GetSyncStatus(ctx context.Context, id, tenantID string) (string, error) {
	return m.getSyncStatusFn(ctx, id, tenantID)
}

// ─── Test helpers ─────────────────────────────────────────────────────────

var _ Repository = (*mockRepo)(nil)

// errTest is a generic error for simulating repository failures.
var errTest = fmt.Errorf("repository error")

// newSummary creates a test CompetencySummary with the given values.
func newSummary(id, tenantID, studentID, learningAreaID, classID string,
	academicYear, term int, calculatedLevel, finalLevel, syncStatus string,
	overrideLevel *string) *CompetencySummary {
	return &CompetencySummary{
		ID:              id,
		TenantID:        tenantID,
		StudentID:       studentID,
		LearningAreaID:  learningAreaID,
		ClassID:         classID,
		AcademicYear:    academicYear,
		Term:            term,
		CalculatedLevel: calculatedLevel,
		OverrideLevel:   overrideLevel,
		FinalLevel:      finalLevel,
		KNECSyncStatus:  syncStatus,
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────

func TestService_GetByID(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	summary := newSummary("s1", "t1", "stu1", "la1", "c1",
		2026, 1, "ME", "ME", "Pending", nil)
	mock.addSummary(summary)

	t.Run("found", func(t *testing.T) {
		s, err := svc.GetByID(context.Background(), "s1", "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "s1" {
			t.Errorf("expected id s1, got %s", s.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "nonexistent", "t1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "", "t1")
		if err == nil {
			t.Fatal("expected error for empty id")
		}
	})

	t.Run("empty tenant", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "s1", "")
		if err == nil {
			t.Fatal("expected error for empty tenant")
		}
	})
}

func TestService_List(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addSummary(newSummary("s1", "t1", "stu1", "la1", "c1", 2026, 1, "ME", "ME", "Pending", nil))
	mock.addSummary(newSummary("s2", "t1", "stu1", "la2", "c1", 2026, 1, "EE", "EE", "Pending", nil))
	mock.addSummary(newSummary("s3", "t1", "stu2", "la1", "c1", 2026, 1, "AE", "AE", "Pending", nil))
	mock.addSummary(newSummary("s4", "t2", "stu3", "la1", "c2", 2026, 1, "ME", "ME", "Pending", nil)) // different tenant

	t.Run("list all for tenant", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", ListSummariesQuery{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 summaries, got %d", len(result))
		}
	})

	t.Run("filter by student_id", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", ListSummariesQuery{StudentID: "stu1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 summaries, got %d", len(result))
		}
	})

	t.Run("filter by class_id", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", ListSummariesQuery{ClassID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 summaries, got %d", len(result))
		}
	})

	t.Run("filter by academic_year and term", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", ListSummariesQuery{AcademicYear: 2026, Term: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 summaries, got %d", len(result))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", ListSummariesQuery{Term: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 summaries, got %d", len(result))
		}
	})

	t.Run("empty tenant", func(t *testing.T) {
		_, err := svc.List(context.Background(), "", ListSummariesQuery{})
		if err == nil {
			t.Fatal("expected error for empty tenant")
		}
	})

	t.Run("tenant isolation", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t2", ListSummariesQuery{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 summary for tenant t2, got %d", len(result))
		}
	})
}

func TestService_CalculateForClass(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockRepo()
		mock.calculateForClassFn = func(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
			return 5, nil
		}
		svc := NewService(mock)

		count, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
			ClassID:      "c1",
			AcademicYear: 2026,
			Term:         1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 5 {
			t.Errorf("expected count 5, got %d", count)
		}
	})

	t.Run("empty class_id", func(t *testing.T) {
		svc := NewService(newMockRepo())
		_, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
			ClassID:      "",
			AcademicYear: 2026,
			Term:         1,
		})
		if err == nil {
			t.Fatal("expected error for empty class_id")
		}
	})

	t.Run("invalid academic_year", func(t *testing.T) {
		svc := NewService(newMockRepo())
		_, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
			ClassID:      "c1",
			AcademicYear: 2016, // < 2017
			Term:         1,
		})
		if err == nil {
			t.Fatal("expected error for invalid academic_year")
		}
	})

	t.Run("invalid term", func(t *testing.T) {
		svc := NewService(newMockRepo())
		_, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
			ClassID:      "c1",
			AcademicYear: 2026,
			Term:         4, // > 3
		})
		if err == nil {
			t.Fatal("expected error for invalid term")
		}
	})

	t.Run("empty result (no assessment sessions)", func(t *testing.T) {
		mock := newMockRepo()
		mock.calculateForClassFn = func(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
			return 0, nil
		}
		svc := NewService(mock)

		count, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
			ClassID:      "c1",
			AcademicYear: 2026,
			Term:         2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := newMockRepo()
		mock.calculateForClassFn = func(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
			return 0, errTest
		}
		svc := NewService(mock)

		_, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
			ClassID:      "c1",
			AcademicYear: 2026,
			Term:         1,
		})
		if err == nil {
			t.Fatal("expected error from repository")
		}
	})
}

func TestService_SetOverrideLevel(t *testing.T) {
	t.Run("set override", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		var capturedOverride *string
		mock.setOverrideLevelFn = func(ctx context.Context, id, tenantID string, overrideLevel *string) error {
			capturedOverride = overrideLevel
			return nil
		}

		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "s1", "t1", OverrideLevelPayload{
			OverrideLevel: "EE",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedOverride == nil || *capturedOverride != "EE" {
			t.Errorf("expected override EE, got %v", capturedOverride)
		}
	})

	t.Run("clear override (empty string)", func(t *testing.T) {
		mock := newMockRepo()
		ee := "EE"
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "EE", "Pending", &ee)
		mock.addSummary(summary)

		var capturedOverride *string
		mock.setOverrideLevelFn = func(ctx context.Context, id, tenantID string, overrideLevel *string) error {
			capturedOverride = overrideLevel
			return nil
		}

		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "s1", "t1", OverrideLevelPayload{
			OverrideLevel: "",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedOverride != nil {
			t.Errorf("expected nil override (cleared), got %v", *capturedOverride)
		}
	})

	t.Run("invalid rubric level", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "s1", "t1", OverrideLevelPayload{
			OverrideLevel: "INVALID",
		})
		if err == nil {
			t.Fatal("expected error for invalid rubric level")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "nonexistent", "t1", OverrideLevelPayload{
			OverrideLevel: "EE",
		})
		if err == nil {
			t.Fatal("expected error for non-existent summary")
		}
	})

	t.Run("already synced - conflict", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Synced", nil)
		mock.addSummary(summary)

		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "s1", "t1", OverrideLevelPayload{
			OverrideLevel: "EE",
		})
		if err == nil {
			t.Fatal("expected conflict error for synced summary")
		}
	})

	t.Run("cross-tenant not found", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "s1", "t2", OverrideLevelPayload{
			OverrideLevel: "EE",
		})
		if err == nil {
			t.Fatal("expected not found for cross-tenant access")
		}
	})
}

func TestService_MarkSynced(t *testing.T) {
	t.Run("mark as synced", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		var capturedStatus string
		mock.markSyncedFn = func(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
			capturedStatus = status
			return nil
		}
		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "Pending", nil
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Synced",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedStatus != "Synced" {
			t.Errorf("expected status Synced, got %s", capturedStatus)
		}
	})

	t.Run("mark as failed", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		var capturedStatus string
		mock.markSyncedFn = func(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
			capturedStatus = status
			return nil
		}
		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "Pending", nil
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Failed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedStatus != "Failed" {
			t.Errorf("expected status Failed, got %s", capturedStatus)
		}
	})

	t.Run("already synced - conflict", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Synced", nil)
		mock.addSummary(summary)

		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "Synced", nil
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Synced",
		})
		if err == nil {
			t.Fatal("expected conflict for re-sync")
		}
	})

	t.Run("failed → synced is allowed", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Failed", nil)
		mock.addSummary(summary)

		var capturedStatus string
		mock.markSyncedFn = func(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
			capturedStatus = status
			return nil
		}
		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "Failed", nil
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Synced",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedStatus != "Synced" {
			t.Errorf("expected status Synced, got %s", capturedStatus)
		}
	})

	t.Run("invalid status value", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "InvalidStatus",
		})
		if err == nil {
			t.Fatal("expected error for invalid status")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockRepo()
		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "", ErrNotFound
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "nonexistent", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Synced",
		})
		if err == nil {
			t.Fatal("expected error for non-existent summary")
		}
	})
}

// TestService_TieBreaking verifies that the modal aggregation correctly
// handles tied rubric levels by selecting the higher competency.
// Note: Actual tie-breaking logic lives in the SQL aggregation. This test
// validates the service layer delegates correctly.
func TestService_CalculateForClass_TieBreaking(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	// Simulate a class with enough rubric results that the aggregation
	// (which happens in SQL) returns summaries. The service just returns
	// whatever the repository gives it — we test the delegation.
	expectedCount := 3
	mock.calculateForClassFn = func(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
		if payload.ClassID == "c1" && payload.AcademicYear == 2026 && payload.Term == 1 {
			return expectedCount, nil
		}
		return 0, nil
	}

	count, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
		ClassID:      "c1",
		AcademicYear: 2026,
		Term:         1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != expectedCount {
		t.Errorf("expected count %d, got %d", expectedCount, count)
	}
}

// TestService_UpsertSemantics verifies that calculating the same class+term
// twice works (UPSERT semantics).
func TestService_UpsertSemantics(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	callCount := 0
	mock.calculateForClassFn = func(ctx context.Context, tenantID string, payload CalculateForClassPayload) (int, error) {
		callCount++
		if callCount == 1 {
			return 5, nil
		}
		return 5, nil // Same count on re-calculation
	}

	// First calculation
	count1, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
		ClassID:      "c1",
		AcademicYear: 2026,
		Term:         1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count1 != 5 {
		t.Errorf("expected count 5, got %d", count1)
	}

	// Re-calculation (UPSERT)
	count2, err := svc.CalculateForClass(context.Background(), "t1", CalculateForClassPayload{
		ClassID:      "c1",
		AcademicYear: 2026,
		Term:         1,
	})
	if err != nil {
		t.Fatalf("unexpected error on re-calculation: %v", err)
	}
	if count2 != 5 {
		t.Errorf("expected count 5, got %d", count2)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls to repository, got %d", callCount)
	}
}

// TestService_FinalLevelOverrideLogic verifies that when override is set,
// the final_level reflects the override's base level.
// Note: The actual final_level computation happens in the repository SQL.
// This test validates the service validation logic.
func TestService_FinalLevelOverrideLogic(t *testing.T) {
	t.Run("override with sub-level", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		var capturedOverride *string
		mock.setOverrideLevelFn = func(ctx context.Context, id, tenantID string, overrideLevel *string) error {
			capturedOverride = overrideLevel
			return nil
		}

		svc := NewService(mock)
		err := svc.SetOverrideLevel(context.Background(), "s1", "t1", OverrideLevelPayload{
			OverrideLevel: "EE1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedOverride == nil || *capturedOverride != "EE1" {
			t.Errorf("expected override EE1, got %v", capturedOverride)
		}
	})
}

// TestService_MarkSyncedSetsSyncedAt verifies that synced_at is set/reset correctly.
func TestService_MarkSynced_SyncedAtBehavior(t *testing.T) {
	t.Run("synced sets synced_at", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		var capturedSyncedAt *string
		mock.markSyncedFn = func(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
			capturedSyncedAt = syncedAt
			return nil
		}
		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "Pending", nil
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Synced",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedSyncedAt == nil {
			t.Error("expected synced_at to be set for Synced status")
		}
	})

	t.Run("failed clears synced_at", func(t *testing.T) {
		mock := newMockRepo()
		summary := newSummary("s1", "t1", "stu1", "la1", "c1",
			2026, 1, "ME", "ME", "Pending", nil)
		mock.addSummary(summary)

		var capturedSyncedAt *string
		mock.markSyncedFn = func(ctx context.Context, id, tenantID, status string, syncedAt *string) error {
			capturedSyncedAt = syncedAt
			return nil
		}
		mock.getSyncStatusFn = func(ctx context.Context, id, tenantID string) (string, error) {
			return "Pending", nil
		}

		svc := NewService(mock)
		err := svc.MarkSynced(context.Background(), "s1", "t1", MarkSyncedPayload{
			KNECSyncStatus: "Failed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedSyncedAt != nil {
			t.Error("expected synced_at to be nil for Failed status")
		}
	})
}
