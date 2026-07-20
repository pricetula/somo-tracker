package reports

import (
	"context"
	"errors"
	"testing"
)

func newServiceWithMocks(
	sp StudentProvider,
	tp TermProvider,
	ap AttendanceProvider,
	asp AssessmentProvider,
	bp BehaviorProvider,
) *Service {
	return NewService(sp, tp, ap, asp, bp)
}

// ============================================================================
// Tests: GetTermReport
// ============================================================================

func TestGetTermReport_HappyPath(t *testing.T) {
	svc := newServiceWithMocks(
		func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
			return &TermReportStudent{
				ID:         "stu_001",
				FullName:   "John Kamau",
				Gender:     "M",
				ClassName:  "G4 East",
				GradeLevel: "G4",
			}, nil
		},
		func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error) {
			return &TermInfo{
				TermID:       "term_001",
				TermName:     "Term 1",
				TermNumber:   1,
				AcademicYear: "2026",
				SchoolName:   "Somo Academy",
			}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error) {
			return []AttendanceSummaryItem{
				{
					LearningAreaID:   "la_001",
					LearningAreaName: "Mathematics",
					PeriodsTotal:     40,
					PeriodsPresent:   38,
					PeriodsAbsent:    1,
					PeriodsLate:      1,
					PeriodsExcused:   0,
					Percentage:       95.0,
				},
				{
					LearningAreaID:   "la_002",
					LearningAreaName: "English",
					PeriodsTotal:     40,
					PeriodsPresent:   35,
					PeriodsAbsent:    3,
					PeriodsLate:      1,
					PeriodsExcused:   1,
					Percentage:       87.5,
				},
			}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*AssessmentData, error) {
			return &AssessmentData{
				LearningAreas: []LearningAreaAssessment{
					{LearningAreaID: "la_001", LearningAreaName: "Mathematics", FinalLevel: "ME", AssessmentCount: 3},
					{LearningAreaID: "la_002", LearningAreaName: "English", FinalLevel: "AE", AssessmentCount: 2},
				},
				Sessions: []AssessmentSessionItem{
					{
						SessionID:        "asmt_001",
						SessionName:      "End Term Exam",
						LearningAreaID:   "la_001",
						EvaluationMethod: "QUANTITATIVE",
						RawScore:         float64Ptr(78),
						MaxPoints:        float64Ptr(100),
						PerformanceLevel: strPtr("ME"),
					},
				},
			}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error) {
			return []BehaviorNoteItem{
				{ID: "beh_001", CategoryName: "Disruptive", Description: "Talked during lesson", Date: "2026-02-15", IsUrgent: false},
			}, nil
		},
	)

	report, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check student
	if report.Student.FullName != "John Kamau" {
		t.Fatalf("expected 'John Kamau', got %q", report.Student.FullName)
	}
	if report.Student.ClassName != "G4 East" {
		t.Fatalf("expected 'G4 East', got %q", report.Student.ClassName)
	}

	// Check term
	if report.Term.TermName != "Term 1" {
		t.Fatalf("expected 'Term 1', got %q", report.Term.TermName)
	}
	if report.Term.SchoolName != "Somo Academy" {
		t.Fatalf("expected 'Somo Academy', got %q", report.Term.SchoolName)
	}

	// Check attendance
	if report.Attendance.OverallPercentage != 91.25 {
		t.Fatalf("expected 91.25 overall attendance, got %f", report.Attendance.OverallPercentage)
	}
	if report.Attendance.PeriodsTotal != 80 {
		t.Fatalf("expected 80 total periods, got %d", report.Attendance.PeriodsTotal)
	}
	if report.Attendance.PeriodsPresent != 73 {
		t.Fatalf("expected 73 present, got %d", report.Attendance.PeriodsPresent)
	}
	if len(report.Attendance.ByLearningArea) != 2 {
		t.Fatalf("expected 2 learning areas, got %d", len(report.Attendance.ByLearningArea))
	}

	// Check assessments
	if len(report.Assessments.LearningAreas) != 2 {
		t.Fatalf("expected 2 assessment learning areas, got %d", len(report.Assessments.LearningAreas))
	}
	if report.Assessments.LearningAreas[0].FinalLevel != "ME" {
		t.Fatalf("expected 'ME', got %q", report.Assessments.LearningAreas[0].FinalLevel)
	}
	if len(report.Assessments.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(report.Assessments.Sessions))
	}

	// Check behavior
	if report.Behavior.TotalIncidents != 1 {
		t.Fatalf("expected 1 behavior incident, got %d", report.Behavior.TotalIncidents)
	}
	if report.Behavior.Notes[0].CategoryName != "Disruptive" {
		t.Fatalf("expected 'Disruptive', got %q", report.Behavior.Notes[0].CategoryName)
	}
}

func TestGetTermReport_EmptyStudentID(t *testing.T) {
	svc := newServiceWithMocks(nil, nil, nil, nil, nil)
	_, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetTermReport_EmptyTermID(t *testing.T) {
	svc := newServiceWithMocks(nil, nil, nil, nil, nil)
	_, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_001", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetTermReport_EmptyTenantID(t *testing.T) {
	svc := newServiceWithMocks(nil, nil, nil, nil, nil)
	_, err := svc.GetTermReport(context.Background(), "", "school_001", "stu_001", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetTermReport_EmptySchoolID(t *testing.T) {
	svc := newServiceWithMocks(nil, nil, nil, nil, nil)
	_, err := svc.GetTermReport(context.Background(), "tenant_001", "", "stu_001", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetTermReport_StudentNotFound(t *testing.T) {
	svc := newServiceWithMocks(
		func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
			return nil, ErrNotFound
		},
		nil, nil, nil, nil,
	)
	_, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_999", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetTermReport_TermNotFound(t *testing.T) {
	svc := newServiceWithMocks(
		func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
			return &TermReportStudent{ID: "stu_001", FullName: "Test"}, nil
		},
		func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error) {
			return nil, ErrNotFound
		},
		nil, nil, nil,
	)
	_, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_001", "term_999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetTermReport_AttendanceFailure(t *testing.T) {
	svc := newServiceWithMocks(
		func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
			return &TermReportStudent{ID: "stu_001", FullName: "Test"}, nil
		},
		func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error) {
			return &TermInfo{TermID: "term_001", TermName: "Term 1"}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error) {
			return nil, errors.New("db connection failed")
		},
		nil, nil,
	)
	_, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTermReport_EmptyAttendance(t *testing.T) {
	svc := newServiceWithMocks(
		func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
			return &TermReportStudent{ID: "stu_001", FullName: "Test"}, nil
		},
		func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error) {
			return &TermInfo{TermID: "term_001", TermName: "Term 1"}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error) {
			return []AttendanceSummaryItem{}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*AssessmentData, error) {
			return &AssessmentData{LearningAreas: []LearningAreaAssessment{}, Sessions: []AssessmentSessionItem{}}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error) {
			return []BehaviorNoteItem{}, nil
		},
	)

	report, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Attendance.OverallPercentage != 0 {
		t.Fatalf("expected 0 attendance with no data, got %f", report.Attendance.OverallPercentage)
	}
	if len(report.Attendance.ByLearningArea) != 0 {
		t.Fatalf("expected 0 learning areas, got %d", len(report.Attendance.ByLearningArea))
	}
	if len(report.Assessments.Sessions) != 0 {
		t.Fatalf("expected 0 assessment sessions, got %d", len(report.Assessments.Sessions))
	}
	if report.Behavior.TotalIncidents != 0 {
		t.Fatalf("expected 0 behavior incidents, got %d", report.Behavior.TotalIncidents)
	}
}

func TestGetTermReport_NilAssessmentData(t *testing.T) {
	svc := newServiceWithMocks(
		func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
			return &TermReportStudent{ID: "stu_001", FullName: "Test"}, nil
		},
		func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error) {
			return &TermInfo{TermID: "term_001", TermName: "Term 1"}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error) {
			return []AttendanceSummaryItem{}, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*AssessmentData, error) {
			return nil, nil
		},
		func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error) {
			return []BehaviorNoteItem{}, nil
		},
	)

	report, err := svc.GetTermReport(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Assessments.LearningAreas == nil {
		t.Fatal("expected non-nil learning areas, got nil")
	}
	if len(report.Assessments.LearningAreas) != 0 {
		t.Fatalf("expected 0 learning areas, got %d", len(report.Assessments.LearningAreas))
	}
}

// Helpers

func float64Ptr(v float64) *float64 {
	return &v
}

func strPtr(s string) *string {
	return &s
}
