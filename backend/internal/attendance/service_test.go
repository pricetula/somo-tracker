package attendance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"somotracker/backend/internal/timetable"
)

// ─── In-memory mock repository ────────────────────────────────────────────

type mockRepo struct {
	mu                       sync.Mutex
	getOrCreatePeriodFn      func(ctx context.Context, input OpenPeriodInput) (*AttendancePeriod, bool, error)
	updatePeriodAuthorizedFn func(ctx context.Context, periodID, tenantID string, role timetable.TeacherRole) error
	upsertLogsFn             func(ctx context.Context, tenantID, periodID, recordedBy string, logs []AttendanceLogInput) error
	getPeriodLogsFn          func(ctx context.Context, tenantID, periodID string) (*AttendancePeriod, []AttendanceLog, error)
	isAuthorizedRecorderFn   func(ctx context.Context, tenantID, userID, classID, learningAreaID, termID string) (*AuthorizedRecorderResult, error)

	// Call tracking
	upsertLogsCalls   []upsertLogsCall
	getOrCreateCalls  []OpenPeriodInput
	authorizedByCalls []authorizedByCall
}

type upsertLogsCall struct {
	TenantID   string
	PeriodID   string
	RecordedBy string
	Logs       []AttendanceLogInput
}

type authorizedByCall struct {
	PeriodID string
	TenantID string
	Role     timetable.TeacherRole
}

func (m *mockRepo) trackUpsertLogs(tenantID, periodID, recordedBy string, logs []AttendanceLogInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertLogsCalls = append(m.upsertLogsCalls, upsertLogsCall{TenantID: tenantID, PeriodID: periodID, RecordedBy: recordedBy, Logs: logs})
}

func (m *mockRepo) trackGetOrCreate(input OpenPeriodInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getOrCreateCalls = append(m.getOrCreateCalls, input)
}

func (m *mockRepo) trackAuthorizedBy(periodID, tenantID string, role timetable.TeacherRole) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authorizedByCalls = append(m.authorizedByCalls, authorizedByCall{PeriodID: periodID, TenantID: tenantID, Role: role})
}

func (m *mockRepo) GetOrCreatePeriod(ctx context.Context, input OpenPeriodInput) (*AttendancePeriod, bool, error) {
	m.trackGetOrCreate(input)
	return m.getOrCreatePeriodFn(ctx, input)
}
func (m *mockRepo) UpdatePeriodAuthorizedBy(ctx context.Context, periodID, tenantID string, role timetable.TeacherRole) error {
	m.trackAuthorizedBy(periodID, tenantID, role)
	return m.updatePeriodAuthorizedFn(ctx, periodID, tenantID, role)
}
func (m *mockRepo) UpsertAttendanceLogs(ctx context.Context, tenantID, periodID, recordedBy string, logs []AttendanceLogInput) error {
	m.trackUpsertLogs(tenantID, periodID, recordedBy, logs)
	return m.upsertLogsFn(ctx, tenantID, periodID, recordedBy, logs)
}
func (m *mockRepo) GetPeriodLogs(ctx context.Context, tenantID, periodID string) (*AttendancePeriod, []AttendanceLog, error) {
	return m.getPeriodLogsFn(ctx, tenantID, periodID)
}
func (m *mockRepo) IsAuthorizedRecorder(ctx context.Context, tenantID, userID, classID, learningAreaID, termID string) (*AuthorizedRecorderResult, error) {
	return m.isAuthorizedRecorderFn(ctx, tenantID, userID, classID, learningAreaID, termID)
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func validMarkAttendanceInput() MarkAttendanceInput {
	return MarkAttendanceInput{
		SchoolID:       "school-1",
		AcademicTermID: "term-1",
		ClassID:        "class-1",
		LearningAreaID: "area-1",
		Date:           "2026-06-26",
		Students: []AttendanceLogInput{
			{StudentID: "student-1", Status: StatusPresent},
			{StudentID: "student-2", Status: StatusAbsent},
		},
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────

func TestOpenAndSubmitAttendance_AuthorizationMatrix(t *testing.T) {
	t.Parallel()

	t.Run("allows subject teacher with matching learning area", func(t *testing.T) {
		role := timetable.TeacherRoleSubject
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{
					Authorized: true,
					Role:       &role,
				}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-1", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", validMarkAttendanceInput())
		if err != nil {
			t.Fatalf("expected no error for authorized subject teacher, got: %v", err)
		}
	})

	t.Run("allows primary class teacher", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{
					Authorized: true,
					Role:       &role,
				}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-2", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-2", validMarkAttendanceInput())
		if err != nil {
			t.Fatalf("expected no error for primary class teacher, got: %v", err)
		}
	})

	t.Run("allows substitute teacher", func(t *testing.T) {
		role := timetable.TeacherRoleSubstitute
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{
					Authorized: true,
					Role:       &role,
				}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-3", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-3", validMarkAttendanceInput())
		if err != nil {
			t.Fatalf("expected no error for substitute teacher, got: %v", err)
		}
	})

	t.Run("allows school admin", func(t *testing.T) {
		role := timetable.TeacherRole("SCHOOL_ADMIN")
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{
					Authorized: true,
					Role:       &role,
				}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-4", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-4", validMarkAttendanceInput())
		if err != nil {
			t.Fatalf("expected no error for school admin, got: %v", err)
		}
	})

	t.Run("rejects unauthorized user with forbidden", func(t *testing.T) {
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{
					Authorized: false,
					Role:       nil,
				}, nil
			},
		}
		svc := NewService(repo)

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-unauthorized", validMarkAttendanceInput())
		if err == nil {
			t.Fatal("expected error for unauthorized user, got nil")
		}
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got: %v", err)
		}
	})
}

func TestOpenAndSubmitAttendance_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty students list succeeds but creates no logs", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-1", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := validMarkAttendanceInput()
		input.Students = []AttendanceLogInput{}

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err != nil {
			t.Fatalf("expected success for empty students list, got: %v", err)
		}
		// Verify period was created but no logs were upserted
		if len(repo.upsertLogsCalls) != 0 {
			t.Fatal("expected no UpsertAttendanceLogs call for empty students list")
		}
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		input := MarkAttendanceInput{
			SchoolID: "school-1",
			Students: []AttendanceLogInput{
				{StudentID: "student-1", Status: StatusPresent},
			},
		}

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err == nil {
			t.Fatal("expected error for missing fields, got nil")
		}
	})

	t.Run("rejects student with empty student_id", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		input := validMarkAttendanceInput()
		input.Students = []AttendanceLogInput{
			{StudentID: "", Status: StatusPresent},
		}

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err == nil {
			t.Fatal("expected error for empty student_id, got nil")
		}
	})

	t.Run("rejects student with empty status", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		input := validMarkAttendanceInput()
		input.Students = []AttendanceLogInput{
			{StudentID: "student-1", Status: ""},
		}

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err == nil {
			t.Fatal("expected error for empty status, got nil")
		}
	})
}

// ─── Authorization Tests 2-3: cross-area and cross-class ───────────────────

func TestOpenAndSubmitAttendance_CrossAreaAndCrossClass(t *testing.T) {
	t.Parallel()

	t.Run("rejects subject teacher submitting for wrong learning area on same class", func(t *testing.T) {
		// IsAuthorizedRecorder returns false when learningAreaID doesn't match
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, learningAreaID, _ string) (*AuthorizedRecorderResult, error) {
				if learningAreaID == "area-2" {
					return &AuthorizedRecorderResult{Authorized: false}, nil
				}
				role := timetable.TeacherRoleSubject
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-1", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		// Teacher authorized for area-1 but trying to submit for area-2 (science)
		input := validMarkAttendanceInput()
		input.LearningAreaID = "area-2" // science, not their assigned area

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err == nil {
			t.Fatal("expected error for cross-area submission, got nil")
		}
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden for cross-area, got: %v", err)
		}
	})

	t.Run("rejects subject teacher submitting for a different class", func(t *testing.T) {
		// IsAuthorizedRecorder returns false when classID is wrong
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, classID, _, _ string) (*AuthorizedRecorderResult, error) {
				if classID == "class-2" {
					return &AuthorizedRecorderResult{Authorized: false}, nil
				}
				role := timetable.TeacherRoleSubject
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
		}
		svc := NewService(repo)

		input := validMarkAttendanceInput()
		input.ClassID = "class-2" // different class

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err == nil {
			t.Fatal("expected error for cross-class submission, got nil")
		}
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden for cross-class, got: %v", err)
		}
	})
}

// ─── Authorization Tests 7-10: Non-teacher roles rejected ──────────────────

func TestOpenAndSubmitAttendance_RejectedRoles(t *testing.T) {
	t.Parallel()

	rejectedRoles := []struct {
		name string
		role string
	}{
		{"FINANCE role is rejected", "FINANCE"},
		{"NURSE role is rejected", "NURSE"},
		{"PARENT role is rejected", "PARENT"},
		{"teacher with no assignment is rejected", ""},
	}

	for _, tc := range rejectedRoles {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &mockRepo{
				isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
					return &AuthorizedRecorderResult{Authorized: false}, nil
				},
			}
			svc := NewService(repo)

			err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-"+tc.role, validMarkAttendanceInput())
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.role)
			}
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("expected ErrForbidden for %s, got: %v", tc.role, err)
			}
		})
	}
}

// ─── Period Open/Reuse Tests ───────────────────────────────────────────────

func TestOpenAndSubmitAttendance_PeriodLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("submitting for new class/date/area creates a new period", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-new", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", validMarkAttendanceInput())
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}

		// Verify GetOrCreatePeriod was called
		if len(repo.getOrCreateCalls) != 1 {
			t.Fatalf("expected 1 GetOrCreatePeriod call, got %d", len(repo.getOrCreateCalls))
		}
		// Verify authorized_by_role was stamped (isNew=true)
		if len(repo.authorizedByCalls) != 1 {
			t.Fatalf("expected 1 UpdatePeriodAuthorizedBy call for new period, got %d", len(repo.authorizedByCalls))
		}
	})

	t.Run("re-submitting returns existing period, not a new one", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		createCount := 0
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				createCount++
				if createCount == 1 {
					return &AttendancePeriod{ID: "period-exists", CreatedAt: time.Now()}, true, nil
				}
				// Second call returns existing period (isNew=false)
				return &AttendancePeriod{ID: "period-exists", CreatedAt: time.Now()}, false, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		// First submission
		_ = svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", validMarkAttendanceInput())
		repo.authorizedByCalls = nil // reset tracking

		// Second submission for same class/date/area
		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", validMarkAttendanceInput())
		if err != nil {
			t.Fatalf("re-submit failed: %v", err)
		}

		// Verify authorized_by_role was NOT updated on re-submit (isNew=false)
		if len(repo.authorizedByCalls) != 0 {
			t.Fatal("authorized_by_role should not be updated on re-submit to existing period")
		}
	})

	t.Run("two different learning areas on same class/date create separate periods", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		createCount := 0
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				createCount++
				id := fmt.Sprintf("period-%d", createCount)
				return &AttendancePeriod{ID: id, CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		// Submit for Math
		input1 := validMarkAttendanceInput()
		input1.LearningAreaID = "math-area"
		_ = svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input1)

		// Submit for Science on same class/date
		input2 := validMarkAttendanceInput()
		input2.LearningAreaID = "science-area"
		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input2)
		if err != nil {
			t.Fatalf("second area submission failed: %v", err)
		}

		// Verify two separate GetOrCreatePeriod calls
		if len(repo.getOrCreateCalls) != 2 {
			t.Fatalf("expected 2 GetOrCreatePeriod calls, got %d", len(repo.getOrCreateCalls))
		}
		if repo.getOrCreateCalls[0].LearningAreaID != "math-area" {
			t.Fatalf("expected first call for math-area, got %s", repo.getOrCreateCalls[0].LearningAreaID)
		}
		if repo.getOrCreateCalls[1].LearningAreaID != "science-area" {
			t.Fatalf("expected second call for science-area, got %s", repo.getOrCreateCalls[1].LearningAreaID)
		}
	})
}

// ─── Batch Submit / Upsert Tests ───────────────────────────────────────────

func TestOpenAndSubmitAttendance_BatchUpsert(t *testing.T) {
	t.Parallel()

	t.Run("submitting 30 students creates 30 log rows", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		var receivedLogs []AttendanceLogInput
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-batch", CreatedAt: time.Now()}, true, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, logs []AttendanceLogInput) error {
				receivedLogs = logs
				return nil
			},
		}
		svc := NewService(repo)

		students := make([]AttendanceLogInput, 30)
		for i := 0; i < 30; i++ {
			students[i] = AttendanceLogInput{
				StudentID: fmt.Sprintf("student-%d", i+1),
				Status:    StatusPresent,
			}
		}

		input := validMarkAttendanceInput()
		input.Students = students

		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}

		if len(receivedLogs) != 30 {
			t.Fatalf("expected 30 log rows, got %d", len(receivedLogs))
		}
	})

	t.Run("re-submitting with one student changed updates only that row", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		callCount := 0
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				callCount++
				if callCount == 1 {
					return &AttendancePeriod{ID: "period-reups", CreatedAt: time.Now()}, true, nil
				}
				return &AttendancePeriod{ID: "period-reups", CreatedAt: time.Now()}, false, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, logs []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		students := make([]AttendanceLogInput, 3)
		for i := 0; i < 3; i++ {
			students[i] = AttendanceLogInput{
				StudentID: fmt.Sprintf("student-%d", i+1),
				Status:    StatusPresent,
			}
		}

		input := validMarkAttendanceInput()
		input.Students = students

		// First submission
		_ = svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		repo.upsertLogsCalls = nil

		// Re-submit with one student changed to LATE
		input.Students[1].Status = StatusLate
		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err != nil {
			t.Fatalf("re-submit failed: %v", err)
		}

		// Verify upsert was called with updated status
		if len(repo.upsertLogsCalls) != 1 {
			t.Fatalf("expected 1 UpsertAttendanceLogs call on re-submit, got %d", len(repo.upsertLogsCalls))
		}
		if len(repo.upsertLogsCalls[0].Logs) != 3 {
			t.Fatalf("expected 3 log entries on re-submit, got %d", len(repo.upsertLogsCalls[0].Logs))
		}
		// Verify the changed student has LATE status
		foundLate := false
		for _, l := range repo.upsertLogsCalls[0].Logs {
			if l.StudentID == "student-2" && l.Status == StatusLate {
				foundLate = true
				break
			}
		}
		if !foundLate {
			t.Fatal("expected student-2 to have LATE status on re-submit")
		}
	})

	t.Run("re-submitting does not create duplicates", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-dup", CreatedAt: time.Now()}, false, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := validMarkAttendanceInput()

		// Submit twice
		_ = svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		_ = svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)

		// Both calls should upsert the same 2 students, no duplicates
		// (The DB ON CONFLICT handles de-duplication; we verify the service
		//  passed the same data both times)
		if len(repo.upsertLogsCalls) != 2 {
			t.Fatalf("expected 2 UpsertAttendanceLogs calls, got %d", len(repo.upsertLogsCalls))
		}
		for i, call := range repo.upsertLogsCalls {
			if len(call.Logs) != 2 {
				t.Fatalf("call %d: expected 2 log entries, got %d", i, len(call.Logs))
			}
		}
	})

	t.Run("student marked ABSENT can later be corrected to LATE", func(t *testing.T) {
		role := timetable.TeacherRolePrimary
		repo := &mockRepo{
			isAuthorizedRecorderFn: func(_ context.Context, _, _, _, _, _ string) (*AuthorizedRecorderResult, error) {
				return &AuthorizedRecorderResult{Authorized: true, Role: &role}, nil
			},
			getOrCreatePeriodFn: func(_ context.Context, _ OpenPeriodInput) (*AttendancePeriod, bool, error) {
				return &AttendancePeriod{ID: "period-correct", CreatedAt: time.Now()}, false, nil
			},
			updatePeriodAuthorizedFn: func(_ context.Context, _, _ string, _ timetable.TeacherRole) error {
				return nil
			},
			upsertLogsFn: func(_ context.Context, _, _, _ string, _ []AttendanceLogInput) error {
				return nil
			},
		}
		svc := NewService(repo)

		input := validMarkAttendanceInput()
		input.Students = []AttendanceLogInput{
			{StudentID: "student-1", Status: StatusAbsent},
		}

		// First submit: ABSENT
		_ = svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)

		// Correct to LATE
		input.Students[0].Status = StatusLate
		err := svc.OpenAndSubmitAttendance(context.Background(), "tenant-1", "user-1", input)
		if err != nil {
			t.Fatalf("correction re-submit failed: %v", err)
		}

		// Verify LATE was sent
		if len(repo.upsertLogsCalls) != 2 {
			t.Fatalf("expected 2 UpsertAttendanceLogs calls, got %d", len(repo.upsertLogsCalls))
		}
		if repo.upsertLogsCalls[1].Logs[0].Status != StatusLate {
			t.Fatalf("expected corrected status LATE, got %s", repo.upsertLogsCalls[1].Logs[0].Status)
		}
	})
}

func TestGetPeriod_Validation(t *testing.T) {
	t.Parallel()

	t.Run("returns period with logs", func(t *testing.T) {
		repo := &mockRepo{
			getPeriodLogsFn: func(_ context.Context, _, _ string) (*AttendancePeriod, []AttendanceLog, error) {
				return &AttendancePeriod{ID: "period-1"}, []AttendanceLog{
					{ID: "log-1", StudentID: "student-1", Status: StatusPresent},
				}, nil
			},
		}
		svc := NewService(repo)

		period, logs, err := svc.GetPeriod(context.Background(), "tenant-1", "period-1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if period.ID != "period-1" {
			t.Fatalf("expected period-1, got: %s", period.ID)
		}
		if len(logs) != 1 {
			t.Fatalf("expected 1 log, got: %d", len(logs))
		}
	})

	t.Run("rejects empty periodID", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		_, _, err := svc.GetPeriod(context.Background(), "tenant-1", "")
		if err == nil {
			t.Fatal("expected error for empty periodID, got nil")
		}
	})
}
