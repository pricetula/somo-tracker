package timetable

import (
	"context"
	"errors"
	"sync"
	"testing"

	"somotracker/backend/internal/middleware"
)

// ─── In-memory mock repository ────────────────────────────────────────────

type mockRepo struct {
	mu               sync.Mutex
	upsertSlotsFn    func(ctx context.Context, tenantID, schoolID string, input BulkCreateTimetableSlotsInput) error
	slotsByClassFn   func(ctx context.Context, tenantID, classID, termID string) ([]TimetableSlot, error)
	slotsByTeacherFn func(ctx context.Context, tenantID, teacherID, termID string) ([]TimetableSlot, error)
	assignTeacherFn  func(ctx context.Context, input ClassTeacherInput) error
	removeTeacherFn  func(ctx context.Context, tenantID, classID, userID string) error
	hasPrimaryRoleFn func(ctx context.Context, tenantID, userID string) (bool, error)
	validateTermFn   func(ctx context.Context, tenantID, schoolID, termID string) (bool, error)
	// Call tracking for tests that need to verify invocations
	assignTeacherCalls []ClassTeacherInput
	removeTeacherCalls []removeTeacherCall
	upsertSlotsCalls   []bulkUpsertCall
}

type removeTeacherCall struct {
	TenantID string
	ClassID  string
	UserID   string
}

type bulkUpsertCall struct {
	TenantID string
	SchoolID string
	Input    BulkCreateTimetableSlotsInput
}

func (m *mockRepo) trackAssignTeacher(input ClassTeacherInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignTeacherCalls = append(m.assignTeacherCalls, input)
}

func (m *mockRepo) trackRemoveTeacher(tenantID, classID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeTeacherCalls = append(m.removeTeacherCalls, removeTeacherCall{TenantID: tenantID, ClassID: classID, UserID: userID})
}

func (m *mockRepo) trackUpsertSlots(tenantID, schoolID string, input BulkCreateTimetableSlotsInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertSlotsCalls = append(m.upsertSlotsCalls, bulkUpsertCall{TenantID: tenantID, SchoolID: schoolID, Input: input})
}

func (m *mockRepo) BulkUpsertSlots(ctx context.Context, tenantID, schoolID string, input BulkCreateTimetableSlotsInput) error {
	m.trackUpsertSlots(tenantID, schoolID, input)
	return m.upsertSlotsFn(ctx, tenantID, schoolID, input)
}
func (m *mockRepo) GetSlotsByClass(ctx context.Context, tenantID, classID, termID string) ([]TimetableSlot, error) {
	return m.slotsByClassFn(ctx, tenantID, classID, termID)
}
func (m *mockRepo) GetSlotsByTeacher(ctx context.Context, tenantID, teacherID, termID string) ([]TimetableSlot, error) {
	return m.slotsByTeacherFn(ctx, tenantID, teacherID, termID)
}
func (m *mockRepo) AssignClassTeacher(ctx context.Context, input ClassTeacherInput) error {
	m.trackAssignTeacher(input)
	return m.assignTeacherFn(ctx, input)
}
func (m *mockRepo) RemoveClassTeacher(ctx context.Context, tenantID, classID, userID string) error {
	m.trackRemoveTeacher(tenantID, classID, userID)
	return m.removeTeacherFn(ctx, tenantID, classID, userID)
}
func (m *mockRepo) HasPrimaryRole(ctx context.Context, tenantID, userID string) (bool, error) {
	return m.hasPrimaryRoleFn(ctx, tenantID, userID)
}
func (m *mockRepo) ValidateTerm(ctx context.Context, tenantID, schoolID, termID string) (bool, error) {
	return m.validateTermFn(ctx, tenantID, schoolID, termID)
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func validPrimaryInput() ClassTeacherInput {
	area := "learning-area-id"
	return ClassTeacherInput{
		TenantID:       "tenant-1",
		SchoolID:       "school-1",
		ClassID:        "class-1",
		UserID:         "user-1",
		LearningAreaID: &area,
		TeacherRole:    TeacherRolePrimary,
	}
}

func TestAssignTeacher_PrimaryUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("rejects second PRIMARY for same teacher", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return true, nil // teacher already has PRIMARY
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := validPrimaryInput()
		input.LearningAreaID = nil // PRIMARY must not have learning area

		err := svc.AssignTeacher(context.Background(), input)
		if err == nil {
			t.Fatal("expected error for duplicate PRIMARY assignment, got nil")
		}
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("expected ErrConflict, got: %v", err)
		}
	})

	t.Run("allows PRIMARY when teacher has no existing PRIMARY", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil // no existing PRIMARY
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := ClassTeacherInput{
			TenantID:       "tenant-1",
			SchoolID:       "school-1",
			ClassID:        "class-1",
			UserID:         "user-1",
			LearningAreaID: nil,
			TeacherRole:    TeacherRolePrimary,
		}

		err := svc.AssignTeacher(context.Background(), input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}

func TestAssignTeacher_SubjectRequiresLearningArea(t *testing.T) {
	t.Parallel()

	t.Run("rejects SUBJECT_TEACHER without learning_area_id", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := ClassTeacherInput{
			TenantID:       "tenant-1",
			SchoolID:       "school-1",
			ClassID:        "class-1",
			UserID:         "user-1",
			LearningAreaID: nil, // missing!
			TeacherRole:    TeacherRoleSubject,
		}

		err := svc.AssignTeacher(context.Background(), input)
		if err == nil {
			t.Fatal("expected error for SUBJECT_TEACHER without learning_area_id, got nil")
		}

		var fe *middleware.FieldError
		if !errors.As(err, &fe) {
			t.Fatalf("expected FieldError, got: %T", err)
		}
		if _, ok := fe.Fields["learning_area_id"]; !ok {
			t.Fatalf("expected field error on 'learning_area_id', got fields: %v", fe.Fields)
		}
	})

	t.Run("rejects PRIMARY_CLASS_TEACHER with learning_area_id", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		area := "some-area"
		input := ClassTeacherInput{
			TenantID:       "tenant-1",
			SchoolID:       "school-1",
			ClassID:        "class-1",
			UserID:         "user-1",
			LearningAreaID: &area, // PRIMARY must not have this!
			TeacherRole:    TeacherRolePrimary,
		}

		err := svc.AssignTeacher(context.Background(), input)
		if err == nil {
			t.Fatal("expected error for PRIMARY with learning_area_id, got nil")
		}

		var fe *middleware.FieldError
		if !errors.As(err, &fe) {
			t.Fatalf("expected FieldError, got: %T", err)
		}
		if _, ok := fe.Fields["learning_area_id"]; !ok {
			t.Fatalf("expected field error on 'learning_area_id', got fields: %v", fe.Fields)
		}
	})

	t.Run("allows SUBJECT_TEACHER with learning_area_id", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		area := "math-area"
		input := ClassTeacherInput{
			TenantID:       "tenant-1",
			SchoolID:       "school-1",
			ClassID:        "class-1",
			UserID:         "user-1",
			LearningAreaID: &area,
			TeacherRole:    TeacherRoleSubject,
		}

		err := svc.AssignTeacher(context.Background(), input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}

func TestAssignTeacher_SubstituteOptionalArea(t *testing.T) {
	t.Parallel()

	t.Run("allows SUBSTITUTE_TEACHER without learning_area_id", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := ClassTeacherInput{
			TenantID:       "tenant-1",
			SchoolID:       "school-1",
			ClassID:        "class-1",
			UserID:         "user-1",
			LearningAreaID: nil,
			TeacherRole:    TeacherRoleSubstitute,
		}

		err := svc.AssignTeacher(context.Background(), input)
		if err != nil {
			t.Fatalf("expected no error for SUBSTITUTE_TEACHER, got: %v", err)
		}
	})

	t.Run("allows SUBSTITUTE_TEACHER with learning_area_id", func(t *testing.T) {
		repo := &mockRepo{
			hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		area := "any-area"
		input := ClassTeacherInput{
			TenantID:       "tenant-1",
			SchoolID:       "school-1",
			ClassID:        "class-1",
			UserID:         "user-1",
			LearningAreaID: &area,
			TeacherRole:    TeacherRoleSubstitute,
		}

		err := svc.AssignTeacher(context.Background(), input)
		if err != nil {
			t.Fatalf("expected no error for SUBSTITUTE_TEACHER, got: %v", err)
		}
	})
}

// ─── Test 4: SUBJECT on A can become PRIMARY on B ────────────────────────

func TestAssignTeacher_SubjectOnA_CanBecomePrimaryOnB(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil // teacher has SUBJECT on A, not PRIMARY
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	input := ClassTeacherInput{
		TenantID:       "tenant-1",
		SchoolID:       "school-1",
		ClassID:        "class-B",
		UserID:         "teacher-1",
		LearningAreaID: nil,
		TeacherRole:    TeacherRolePrimary,
	}

	err := svc.AssignTeacher(context.Background(), input)
	if err != nil {
		t.Fatalf("expected SUBJECT teacher to be assignable as PRIMARY on another class, got: %v", err)
	}
}

// ─── Test 5: Two different teachers can both be PRIMARY on different classes ─

func TestAssignTeacher_TwoTeachersPrimaryDifferentClasses(t *testing.T) {
	t.Parallel()

	callCount := 0
	repo := &mockRepo{
		hasPrimaryRoleFn: func(_ context.Context, _, userID string) (bool, error) {
			// Both teachers have no existing PRIMARY
			return false, nil
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			callCount++
			return nil
		},
	}
	svc := NewService(repo)

	// Teacher A -> Class A
	err := svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-A",
		UserID: "teacher-A", LearningAreaID: nil, TeacherRole: TeacherRolePrimary,
	})
	if err != nil {
		t.Fatalf("first teacher assignment failed: %v", err)
	}

	// Teacher B -> Class B
	err = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-B",
		UserID: "teacher-B", LearningAreaID: nil, TeacherRole: TeacherRolePrimary,
	})
	if err != nil {
		t.Fatalf("second teacher assignment failed: %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected 2 repo calls, got %d", callCount)
	}
}

// ─── Test 8: Same teacher SUBJECT for two learning areas on same class ──────

func TestAssignTeacher_SameTeacherTwoSubjectsSameClass(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	area1 := "math"
	area2 := "science"

	// Assign teacher as SUBJECT for Math
	err := svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-1",
		UserID: "teacher-1", LearningAreaID: &area1, TeacherRole: TeacherRoleSubject,
	})
	if err != nil {
		t.Fatalf("first subject assignment failed: %v", err)
	}

	// Assign same teacher as SUBJECT for Science on same class
	err = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-1",
		UserID: "teacher-1", LearningAreaID: &area2, TeacherRole: TeacherRoleSubject,
	})
	if err != nil {
		t.Fatalf("second subject assignment on same class failed: %v", err)
	}
}

// ─── Test 9: Two teachers SUBJECT for same area on different classes ────────

func TestAssignTeacher_TwoTeachersSubjectSameAreaDifferentClasses(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	area := "math"

	// Teacher A -> Class A for Math
	err := svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-A",
		UserID: "teacher-A", LearningAreaID: &area, TeacherRole: TeacherRoleSubject,
	})
	if err != nil {
		t.Fatalf("first teacher assignment failed: %v", err)
	}

	// Teacher B -> Class B for Math
	err = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-B",
		UserID: "teacher-B", LearningAreaID: &area, TeacherRole: TeacherRoleSubject,
	})
	if err != nil {
		t.Fatalf("second teacher assignment failed: %v", err)
	}
}

// ─── Test 13: SUBSTITUTE on multiple classes simultaneously ─────────────────

func TestAssignTeacher_SubstituteOnMultipleClasses(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	// Same teacher as SUBSTITUTE on Class A
	err := svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-A",
		UserID: "teacher-1", LearningAreaID: nil, TeacherRole: TeacherRoleSubstitute,
	})
	if err != nil {
		t.Fatalf("first substitute assignment failed: %v", err)
	}

	// Same teacher as SUBSTITUTE on Class B
	err = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-B",
		UserID: "teacher-1", LearningAreaID: nil, TeacherRole: TeacherRoleSubstitute,
	})
	if err != nil {
		t.Fatalf("second substitute assignment on different class failed: %v", err)
	}
}

// ─── Test 14: PRIMARY on Class A can be SUBJECT on Class B ──────────────────

func TestAssignTeacher_PrimaryOnA_CanBeSubjectOnB(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	area := "science"

	// Assign as PRIMARY on Class A
	err := svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-A",
		UserID: "teacher-1", LearningAreaID: nil, TeacherRole: TeacherRolePrimary,
	})
	if err != nil {
		t.Fatalf("PRIMARY assignment failed: %v", err)
	}

	// Assign same teacher as SUBJECT on Class B
	err = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-B",
		UserID: "teacher-1", LearningAreaID: &area, TeacherRole: TeacherRoleSubject,
	})
	if err != nil {
		t.Fatalf("SUBJECT assignment on different class failed: %v", err)
	}
}

// ─── Test 16: Removing teacher from one class does not affect other classes ──

func TestRemoveTeacher_ScopedToSingleClass(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		removeTeacherFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		assignTeacherFn: func(_ context.Context, _ ClassTeacherInput) error {
			return nil
		},
		hasPrimaryRoleFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		upsertSlotsFn: func(_ context.Context, _, _ string, _ BulkCreateTimetableSlotsInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	// Assign teacher to Class A
	_ = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-A",
		UserID: "teacher-1", LearningAreaID: nil, TeacherRole: TeacherRolePrimary,
	})
	// Assign teacher to Class B
	area := "math"
	_ = svc.AssignTeacher(context.Background(), ClassTeacherInput{
		TenantID: "tenant-1", SchoolID: "school-1", ClassID: "class-B",
		UserID: "teacher-1", LearningAreaID: &area, TeacherRole: TeacherRoleSubject,
	})

	// Clear call tracking from setup
	repo.assignTeacherCalls = nil

	// Remove teacher ONLY from Class A
	err := svc.RemoveTeacher(context.Background(), "tenant-1", "class-A", "teacher-1")
	if err != nil {
		t.Fatalf("RemoveTeacher failed: %v", err)
	}

	// Verify RemoveTeacher was called with correct (tenant-1, class-A, teacher-1)
	if len(repo.removeTeacherCalls) != 1 {
		t.Fatalf("expected 1 RemoveTeacher call, got %d", len(repo.removeTeacherCalls))
	}
	call := repo.removeTeacherCalls[0]
	if call.TenantID != "tenant-1" || call.ClassID != "class-A" || call.UserID != "teacher-1" {
		t.Fatalf("RemoveTeacher called with wrong params: %+v", call)
	}
}

// ─── Bulk Slot Save: Successful bulk insert ─────────────────────────────────

func TestBulkSaveSlots_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		validateTermFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return true, nil
		},
		upsertSlotsFn: func(_ context.Context, _, _ string, _ BulkCreateTimetableSlotsInput) error {
			return nil
		},
	}
	svc := NewService(repo)

	input := BulkCreateTimetableSlotsInput{
		AcademicYearID: "year-1",
		AcademicTermID: "term-1",
		Slots: []CreateTimetableSlotInput{
			{ClassID: "class-1", TeacherID: "teacher-1", DayOfWeek: 1, StartTime: "08:00", EndTime: "09:00"},
			{ClassID: "class-1", TeacherID: "teacher-2", DayOfWeek: 1, StartTime: "09:00", EndTime: "10:00"},
			{ClassID: "class-2", TeacherID: "teacher-1", DayOfWeek: 2, StartTime: "08:00", EndTime: "09:00"},
		},
	}

	err := svc.BulkSaveSlots(context.Background(), "tenant-1", "school-1", input)
	if err != nil {
		t.Fatalf("expected successful bulk save, got: %v", err)
	}

	if len(repo.upsertSlotsCalls) != 1 {
		t.Fatalf("expected 1 BulkUpsertSlots call, got %d", len(repo.upsertSlotsCalls))
	}
	if len(repo.upsertSlotsCalls[0].Input.Slots) != 3 {
		t.Fatalf("expected 3 slots in repo call, got %d", len(repo.upsertSlotsCalls[0].Input.Slots))
	}
}

// ─── Bulk Slot Save: Repo error propagates ──────────────────────────────────

func TestBulkSaveSlots_RepoErrorPropagates(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		validateTermFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return true, nil
		},
		upsertSlotsFn: func(_ context.Context, _, _ string, _ BulkCreateTimetableSlotsInput) error {
			return ErrConflict
		},
	}
	svc := NewService(repo)

	input := BulkCreateTimetableSlotsInput{
		AcademicYearID: "year-1",
		AcademicTermID: "term-1",
		Slots: []CreateTimetableSlotInput{
			{ClassID: "class-1", TeacherID: "teacher-1", DayOfWeek: 1, StartTime: "08:00", EndTime: "09:00"},
		},
	}

	err := svc.BulkSaveSlots(context.Background(), "tenant-1", "school-1", input)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}
}

func TestBulkSaveSlots_Validation(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty slots", func(t *testing.T) {
		repo := &mockRepo{
			validateTermFn: func(_ context.Context, _, _, _ string) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		input := BulkCreateTimetableSlotsInput{
			AcademicYearID: "year-1",
			AcademicTermID: "term-1",
			Slots:          []CreateTimetableSlotInput{},
		}

		err := svc.BulkSaveSlots(context.Background(), "tenant-1", "school-1", input)
		if err == nil {
			t.Fatal("expected error for empty slots, got nil")
		}
	})

	t.Run("rejects slot with missing class_id", func(t *testing.T) {
		repo := &mockRepo{
			validateTermFn: func(_ context.Context, _, _, _ string) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		input := BulkCreateTimetableSlotsInput{
			AcademicYearID: "year-1",
			AcademicTermID: "term-1",
			Slots: []CreateTimetableSlotInput{
				{
					TeacherID: "teacher-1",
					DayOfWeek: 1,
					StartTime: "08:00",
					EndTime:   "09:00",
				},
			},
		}

		err := svc.BulkSaveSlots(context.Background(), "tenant-1", "school-1", input)
		if err == nil {
			t.Fatal("expected error for missing class_id, got nil")
		}
	})
}
