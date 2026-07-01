package parents

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// ─── In-memory mock repository ────────────────────────────────────────────

type mockRepo struct {
	mu sync.Mutex

	parents  map[string]*Parent       // keyed by ID
	links    map[string][]StudentLink // parentID -> linked students
	students map[string]bool          // studentID -> exists (tenant-specific)
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		parents:  make(map[string]*Parent),
		links:    make(map[string][]StudentLink),
		students: make(map[string]bool),
	}
}

// addStudent registers a student as existing in a tenant.
func (m *mockRepo) addStudent(studentID, tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.students[studentID+"|"+tenantID] = true
}

func (m *mockRepo) addParent(p *Parent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parents[p.ID] = p
	m.links[p.ID] = []StudentLink{}
}

func (m *mockRepo) StudentExistsInTenant(ctx context.Context, studentID, tenantID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.students[studentID+"|"+tenantID], nil
}

func (m *mockRepo) Create(ctx context.Context, tenantID string, payload CreateParentPayload) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate
	for _, p := range m.parents {
		if p.Email == payload.Email && p.TenantID == tenantID {
			return "", ErrAlreadyExists
		}
	}

	id := fmt.Sprintf("parent_%d", len(m.parents)+1)
	p := &Parent{
		ID:          id,
		TenantID:    tenantID,
		UserID:      "user_" + id,
		FullName:    payload.FullName,
		Email:       payload.Email,
		PhoneNumber: payload.PhoneNumber,
		IsActive:    true,
		CreatedAt:   "2026-07-01T00:00:00Z",
	}
	m.parents[id] = p
	m.links[id] = []StudentLink{}
	return id, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id, tenantID string) (*Parent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parents[id]
	if !ok {
		return nil, ErrNotFound
	}
	if p.TenantID != tenantID {
		return nil, ErrNotFound
	}
	pCopy := *p
	return &pCopy, nil
}

func (m *mockRepo) GetDetail(ctx context.Context, id, tenantID string) (*ParentDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parents[id]
	if !ok {
		return nil, ErrNotFound
	}
	if p.TenantID != tenantID {
		return nil, ErrNotFound
	}
	pCopy := *p
	links := m.links[id]
	if links == nil {
		links = []StudentLink{}
	}
	linksCopy := make([]StudentLink, len(links))
	copy(linksCopy, links)
	return &ParentDetail{
		Parent:         pCopy,
		LinkedStudents: linksCopy,
	}, nil
}

func (m *mockRepo) List(ctx context.Context, tenantID string, search, studentID string) ([]Parent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []Parent
	for _, p := range m.parents {
		if p.TenantID != tenantID {
			continue
		}
		if search != "" {
			// Simple substring match
			if !containsStr(p.FullName, search) && !containsStr(p.Email, search) {
				continue
			}
		}
		if studentID != "" {
			// Check if linked to this student
			links := m.links[p.ID]
			found := false
			for _, l := range links {
				if l.StudentID == studentID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, *p)
	}
	if result == nil {
		result = []Parent{}
	}
	return result, nil
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStrInner(s, substr))
}

func containsStrInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (m *mockRepo) Update(ctx context.Context, id, tenantID string, payload UpdateParentPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parents[id]
	if !ok {
		return ErrNotFound
	}
	if p.TenantID != tenantID {
		return ErrNotFound
	}
	if payload.PhoneNumber != nil {
		p.PhoneNumber = *payload.PhoneNumber
	}
	if payload.IsActive != nil {
		p.IsActive = *payload.IsActive
	}
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parents[id]
	if !ok {
		return ErrNotFound
	}
	if p.TenantID != tenantID {
		return ErrNotFound
	}
	delete(m.parents, id)
	delete(m.links, id)
	return nil
}

func (m *mockRepo) LinkStudent(ctx context.Context, parentID, tenantID string, payload LinkStudentPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify parent exists and belongs to tenant
	p, ok := m.parents[parentID]
	if !ok || p.TenantID != tenantID {
		return ErrNotFound
	}

	// Verify student exists in tenant
	if !m.students[payload.StudentID+"|"+tenantID] {
		return ErrStudentNotFound
	}

	// Check for duplicate
	for _, l := range m.links[parentID] {
		if l.StudentID == payload.StudentID {
			return ErrDuplicateLink
		}
	}

	// If is_primary, demote others for this student
	isPrimary := false
	if payload.IsPrimary != nil {
		isPrimary = *payload.IsPrimary
	}
	if isPrimary {
		for pid, links := range m.links {
			for i, l := range links {
				if l.StudentID == payload.StudentID && l.IsPrimary {
					m.links[pid][i] = StudentLink{
						StudentID:    l.StudentID,
						FullName:     l.FullName,
						Relationship: l.Relationship,
						IsPrimary:    false,
					}
				}
			}
		}
	}

	link := StudentLink{
		StudentID:    payload.StudentID,
		FullName:     "Student " + payload.StudentID,
		Relationship: payload.Relationship,
		IsPrimary:    isPrimary,
	}
	m.links[parentID] = append(m.links[parentID], link)
	return nil
}

func (m *mockRepo) UnlinkStudent(ctx context.Context, parentID, studentID, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parents[parentID]
	if !ok || p.TenantID != tenantID {
		return ErrNotFound
	}
	links := m.links[parentID]
	for i, l := range links {
		if l.StudentID == studentID {
			m.links[parentID] = append(links[:i], links[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) DemotePrimaryForStudent(ctx context.Context, studentID, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for pid, links := range m.links {
		for i, l := range links {
			if l.StudentID == studentID && l.IsPrimary {
				m.links[pid][i] = StudentLink{
					StudentID:    l.StudentID,
					FullName:     l.FullName,
					Relationship: l.Relationship,
					IsPrimary:    false,
				}
			}
		}
	}
	return nil
}

func (m *mockRepo) CountLinksByStudent(ctx context.Context, studentID, tenantID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for pid, links := range m.links {
		if p, ok := m.parents[pid]; ok && p.TenantID == tenantID {
			for _, l := range links {
				if l.StudentID == studentID {
					count++
				}
			}
		}
	}
	return count, nil
}

// Ensure mockRepo implements Repository
var _ Repository = (*mockRepo)(nil)

// ─── Test helpers ─────────────────────────────────────────────────────────

func newParent(id, tenantID, email, fullName, phone string) *Parent {
	return &Parent{
		ID:          id,
		TenantID:    tenantID,
		UserID:      "user_" + id,
		FullName:    fullName,
		Email:       email,
		PhoneNumber: phone,
		IsActive:    true,
		CreatedAt:   "2026-07-01T00:00:00Z",
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────

func TestService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)

		parent, err := svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "parent@example.com",
			FullName:    "John Doe",
			PhoneNumber: "+254712345678",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parent.ID == "" {
			t.Error("expected non-empty ID")
		}
		if parent.Email != "parent@example.com" {
			t.Errorf("expected email parent@example.com, got %s", parent.Email)
		}
		if !parent.IsActive {
			t.Error("expected parent to be active by default")
		}
	})

	t.Run("empty email", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)

		_, err := svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "",
			FullName:    "John Doe",
			PhoneNumber: "+254712345678",
		})
		if err == nil {
			t.Fatal("expected error for empty email")
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)

		_, err := svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "not-an-email",
			FullName:    "John Doe",
			PhoneNumber: "+254712345678",
		})
		if err == nil {
			t.Fatal("expected error for invalid email")
		}
	})

	t.Run("empty phone number", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)

		_, err := svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "parent@example.com",
			FullName:    "John Doe",
			PhoneNumber: "",
		})
		if err == nil {
			t.Fatal("expected error for empty phone_number")
		}
	})

	t.Run("empty full_name", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)

		_, err := svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "parent@example.com",
			FullName:    "",
			PhoneNumber: "+254712345678",
		})
		if err == nil {
			t.Fatal("expected error for empty full_name")
		}
	})

	t.Run("duplicate profile (same email in tenant)", func(t *testing.T) {
		mock := newMockRepo()
		svc := NewService(mock)

		// Create first parent
		_, err := svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "parent@example.com",
			FullName:    "John Doe",
			PhoneNumber: "+254712345678",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Attempt duplicate
		_, err = svc.Create(context.Background(), "t1", CreateParentPayload{
			Email:       "parent@example.com",
			FullName:    "John Doe",
			PhoneNumber: "+254712345678",
		})
		if err == nil {
			t.Fatal("expected error for duplicate parent profile")
		}
	})
}

func TestService_GetByID(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))

	t.Run("found", func(t *testing.T) {
		p, err := svc.GetByID(context.Background(), "p1", "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != "p1" {
			t.Errorf("expected id p1, got %s", p.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "nonexistent", "t1")
		if err == nil {
			t.Fatal("expected error for non-existent parent")
		}
	})

	t.Run("cross-tenant", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "p1", "t2")
		if err == nil {
			t.Fatal("expected error for cross-tenant access")
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "", "t1")
		if err == nil {
			t.Fatal("expected error for empty id")
		}
	})
}

func TestService_GetDetail(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))
	mock.addStudent("s1", "t1")

	// Link a student
	err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
		StudentID: "s1",
		IsPrimary: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("unexpected error linking student: %v", err)
	}

	t.Run("detail with linked students", func(t *testing.T) {
		detail, err := svc.GetDetail(context.Background(), "p1", "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(detail.LinkedStudents) != 1 {
			t.Errorf("expected 1 linked student, got %d", len(detail.LinkedStudents))
		}
		if detail.LinkedStudents[0].StudentID != "s1" {
			t.Errorf("expected student s1, got %s", detail.LinkedStudents[0].StudentID)
		}
		if !detail.LinkedStudents[0].IsPrimary {
			t.Error("expected linked student to be primary")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetDetail(context.Background(), "nonexistent", "t1")
		if err == nil {
			t.Fatal("expected error for non-existent parent")
		}
	})
}

func TestService_List(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "alice@example.com", "Alice Parent", "+254700000001"))
	mock.addParent(newParent("p2", "t1", "bob@example.com", "Bob Parent", "+254700000002"))
	mock.addParent(newParent("p3", "t2", "carol@example.com", "Carol Parent", "+254700000003")) // different tenant

	t.Run("list all for tenant", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 parents, got %d", len(result))
		}
	})

	t.Run("search by name", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", "Alice", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 parent, got %d", len(result))
		}
		if result[0].ID != "p1" {
			t.Errorf("expected p1, got %s", result[0].ID)
		}
	})

	t.Run("search by email", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t1", "bob@example.com", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 parent, got %d", len(result))
		}
	})

	t.Run("filter by student_id", func(t *testing.T) {
		mock.addStudent("s1", "t1")
		if err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
			StudentID: "s1",
		}); err != nil {
			t.Fatalf("unexpected error linking student: %v", err)
		}

		result, err := svc.List(context.Background(), "t1", "", "s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 parent linked to s1, got %d", len(result))
		}
	})

	t.Run("tenant isolation", func(t *testing.T) {
		result, err := svc.List(context.Background(), "t2", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 parent for tenant t2, got %d", len(result))
		}
	})

	t.Run("empty tenant", func(t *testing.T) {
		_, err := svc.List(context.Background(), "", "", "")
		if err == nil {
			t.Fatal("expected error for empty tenant")
		}
	})
}

func TestService_Update(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))

	t.Run("update phone number", func(t *testing.T) {
		newPhone := "+254711111111"
		err := svc.Update(context.Background(), "p1", "t1", UpdateParentPayload{
			PhoneNumber: &newPhone,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p, _ := svc.GetByID(context.Background(), "p1", "t1")
		if p.PhoneNumber != newPhone {
			t.Errorf("expected phone %s, got %s", newPhone, p.PhoneNumber)
		}
	})

	t.Run("deactivate parent", func(t *testing.T) {
		inactive := false
		err := svc.Update(context.Background(), "p1", "t1", UpdateParentPayload{
			IsActive: &inactive,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p, _ := svc.GetByID(context.Background(), "p1", "t1")
		if p.IsActive {
			t.Error("expected parent to be inactive")
		}
	})

	t.Run("not found", func(t *testing.T) {
		phone := "+254722222222"
		err := svc.Update(context.Background(), "nonexistent", "t1", UpdateParentPayload{
			PhoneNumber: &phone,
		})
		if err == nil {
			t.Fatal("expected error for non-existent parent")
		}
	})

	t.Run("no fields provided", func(t *testing.T) {
		err := svc.Update(context.Background(), "p1", "t1", UpdateParentPayload{})
		if err == nil {
			t.Fatal("expected error when no fields provided")
		}
	})

	t.Run("cross-tenant", func(t *testing.T) {
		phone := "+254733333333"
		err := svc.Update(context.Background(), "p1", "t2", UpdateParentPayload{
			PhoneNumber: &phone,
		})
		if err == nil {
			t.Fatal("expected error for cross-tenant")
		}
	})
}

func TestService_Delete(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))

	t.Run("success", func(t *testing.T) {
		err := svc.Delete(context.Background(), "p1", "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deleted
		_, err = svc.GetByID(context.Background(), "p1", "t1")
		if err == nil {
			t.Fatal("expected parent to be deleted")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(context.Background(), "nonexistent", "t1")
		if err == nil {
			t.Fatal("expected error for non-existent parent")
		}
	})
}

func TestService_LinkStudent(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))
	mock.addStudent("s1", "t1")

	t.Run("success", func(t *testing.T) {
		err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
			StudentID:    "s1",
			Relationship: strPtr("Father"),
			IsPrimary:    boolPtr(true),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detail, _ := svc.GetDetail(context.Background(), "p1", "t1")
		if len(detail.LinkedStudents) != 1 {
			t.Fatalf("expected 1 linked student, got %d", len(detail.LinkedStudents))
		}
		if detail.LinkedStudents[0].Relationship == nil || *detail.LinkedStudents[0].Relationship != "Father" {
			t.Errorf("expected relationship Father, got %v", detail.LinkedStudents[0].Relationship)
		}
	})

	t.Run("duplicate link", func(t *testing.T) {
		err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
			StudentID: "s1",
		})
		if err == nil {
			t.Fatal("expected error for duplicate link")
		}
	})

	t.Run("non-existent student", func(t *testing.T) {
		err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
			StudentID: "nonexistent",
		})
		if err == nil {
			t.Fatal("expected error for non-existent student")
		}
	})

	t.Run("non-existent parent", func(t *testing.T) {
		err := svc.LinkStudent(context.Background(), "nonexistent", "t1", LinkStudentPayload{
			StudentID: "s1",
		})
		if err == nil {
			t.Fatal("expected error for non-existent parent")
		}
	})

	t.Run("empty student_id", func(t *testing.T) {
		err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
			StudentID: "",
		})
		if err == nil {
			t.Fatal("expected error for empty student_id")
		}
	})
}

func TestService_LinkStudent_PrimaryDemotion(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))
	mock.addParent(newParent("p2", "t1", "p2@example.com", "Parent Two", "+254700000002"))
	mock.addStudent("s1", "t1")

	// Link first parent as primary
	err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
		StudentID: "s1",
		IsPrimary: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Link second parent also as primary — should demote p1
	err = svc.LinkStudent(context.Background(), "p2", "t1", LinkStudentPayload{
		StudentID: "s1",
		IsPrimary: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check p1's link is no longer primary
	p1Detail, _ := svc.GetDetail(context.Background(), "p1", "t1")
	if len(p1Detail.LinkedStudents) != 1 {
		t.Fatalf("expected 1 linked student, got %d", len(p1Detail.LinkedStudents))
	}
	if p1Detail.LinkedStudents[0].IsPrimary {
		t.Error("expected p1's link to be demoted from primary")
	}

	// Check p2's link is primary
	p2Detail, _ := svc.GetDetail(context.Background(), "p2", "t1")
	if len(p2Detail.LinkedStudents) != 1 {
		t.Fatalf("expected 1 linked student, got %d", len(p2Detail.LinkedStudents))
	}
	if !p2Detail.LinkedStudents[0].IsPrimary {
		t.Error("expected p2's link to be primary")
	}
}

func TestService_UnlinkStudent(t *testing.T) {
	mock := newMockRepo()
	svc := NewService(mock)

	mock.addParent(newParent("p1", "t1", "p1@example.com", "Parent One", "+254700000001"))
	mock.addStudent("s1", "t1")

	// Link first
	if err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
		StudentID: "s1",
	}); err != nil {
		t.Fatalf("unexpected error linking student: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		err := svc.UnlinkStudent(context.Background(), "p1", "s1", "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detail, _ := svc.GetDetail(context.Background(), "p1", "t1")
		if len(detail.LinkedStudents) != 0 {
			t.Errorf("expected 0 linked students after unlink, got %d", len(detail.LinkedStudents))
		}
	})

	t.Run("not found (already unlinked)", func(t *testing.T) {
		err := svc.UnlinkStudent(context.Background(), "p1", "s1", "t1")
		if err == nil {
			t.Fatal("expected error for already unlinked student")
		}
	})

	t.Run("non-existent parent", func(t *testing.T) {
		err := svc.UnlinkStudent(context.Background(), "nonexistent", "s1", "t1")
		if err == nil {
			t.Fatal("expected error for non-existent parent")
		}
	})

	t.Run("empty params", func(t *testing.T) {
		err := svc.UnlinkStudent(context.Background(), "", "", "")
		if err == nil {
			t.Fatal("expected error for empty params")
		}
	})

	t.Run("cross-tenant", func(t *testing.T) {
		// Re-link first
		if err := svc.LinkStudent(context.Background(), "p1", "t1", LinkStudentPayload{
			StudentID: "s1",
		}); err != nil {
			t.Fatalf("unexpected error linking student: %v", err)
		}
		err := svc.UnlinkStudent(context.Background(), "p1", "s1", "t2")
		if err == nil {
			t.Fatal("expected error for cross-tenant unlink")
		}
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
