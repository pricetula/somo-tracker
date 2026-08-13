package reports

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Service orchestrates data from attendance, assessments, and behavior domains
// to produce compiled CBC term report cards.
type Service struct {
	studentProvider    StudentProvider
	termProvider       TermProvider
	attendanceProvider AttendanceProvider
	assessmentProvider AssessmentProvider
	behaviorProvider   BehaviorProvider
	logger             *zap.SugaredLogger
}

// NewService creates a new reports orchestrator Service.
func NewService(
	sp StudentProvider,
	tp TermProvider,
	ap AttendanceProvider,
	asp AssessmentProvider,
	bp BehaviorProvider,
	logger *zap.SugaredLogger,
) *Service {
	return &Service{
		studentProvider:    sp,
		termProvider:       tp,
		attendanceProvider: ap,
		assessmentProvider: asp,
		behaviorProvider:   bp,
		logger:             logger,
	}
}

// GetTermReport compiles a full term report card for a single student.
func (s *Service) GetTermReport(ctx context.Context, tenantID, schoolID, studentID, termID string) (*TermReport, error) {
	if studentID == "" {
		return nil, fmt.Errorf("reports.Service.GetTermReport: student_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("reports.Service.GetTermReport: term_id is required: %w", ErrInvalidInput)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("reports.Service.GetTermReport: tenant_id is required: %w", ErrInvalidInput)
	}
	if schoolID == "" {
		return nil, fmt.Errorf("reports.Service.GetTermReport: school_id is required: %w", ErrInvalidInput)
	}

	// Fetch all data sources concurrently.
	type studentResult struct {
		student *TermReportStudent
		err     error
	}
	studentCh := make(chan studentResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warnw("reports.Service.GetTermReport: student goroutine panicked", "panic", r)
			}
		}()
		s, err := s.studentProvider(ctx, studentID, tenantID, schoolID)
		studentCh <- studentResult{student: s, err: err}
	}()

	type termResult struct {
		term *TermInfo
		err  error
	}
	termCh := make(chan termResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warnw("reports.Service.GetTermReport: term goroutine panicked", "panic", r)
			}
		}()
		t, err := s.termProvider(ctx, termID, tenantID, schoolID)
		termCh <- termResult{term: t, err: err}
	}()

	type attendanceResult struct {
		items []AttendanceSummaryItem
		err   error
	}
	attCh := make(chan attendanceResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warnw("reports.Service.GetTermReport: attendance goroutine panicked", "panic", r)
			}
		}()
		items, err := s.attendanceProvider(ctx, tenantID, schoolID, studentID, termID)
		attCh <- attendanceResult{items: items, err: err}
	}()

	type assessmentResult struct {
		data *AssessmentData
		err  error
	}
	asmtCh := make(chan assessmentResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warnw("reports.Service.GetTermReport: assessment goroutine panicked", "panic", r)
			}
		}()
		data, err := s.assessmentProvider(ctx, tenantID, schoolID, studentID, termID)
		asmtCh <- assessmentResult{data: data, err: err}
	}()

	type behaviorResult struct {
		notes []BehaviorNoteItem
		err   error
	}
	behCh := make(chan behaviorResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warnw("reports.Service.GetTermReport: behavior goroutine panicked", "panic", r)
			}
		}()
		notes, err := s.behaviorProvider(ctx, tenantID, schoolID, studentID, termID)
		behCh <- behaviorResult{notes: notes, err: err}
	}()

	// Collect results.
	sr := <-studentCh
	if sr.err != nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: student: %w", sr.err)
	}
	if sr.student == nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: %w", ErrNotFound)
	}

	tr := <-termCh
	if tr.err != nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: term: %w", tr.err)
	}
	if tr.term == nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: %w", ErrNotFound)
	}

	ar := <-attCh
	if ar.err != nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: attendance: %w", ar.err)
	}

	asr := <-asmtCh
	if asr.err != nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: assessments: %w", asr.err)
	}

	br := <-behCh
	if br.err != nil {
		return nil, fmt.Errorf("reports.Service.GetTermReport: behavior: %w", br.err)
	}

	// Compute overall attendance rollup.
	attSection := s.buildAttendanceSection(ar.items)
	asmtSection := s.buildAssessmentSection(asr.data)
	behSection := s.buildBehaviorSection(br.notes)

	report := &TermReport{
		Student:     *sr.student,
		Term:        *tr.term,
		Attendance:  attSection,
		Assessments: asmtSection,
		Behavior:    behSection,
	}

	return report, nil
}

func (s *Service) buildAttendanceSection(items []AttendanceSummaryItem) AttendanceSection {
	if items == nil {
		items = []AttendanceSummaryItem{}
	}

	total := 0
	present := 0
	absent := 0
	late := 0
	excused := 0

	for _, item := range items {
		total += item.PeriodsTotal
		present += item.PeriodsPresent
		absent += item.PeriodsAbsent
		late += item.PeriodsLate
		excused += item.PeriodsExcused
	}

	overallPct := 0.0
	if total > 0 {
		overallPct = float64(present) / float64(total) * 100
	}

	las := make([]LearningAreaAttendance, len(items))
	for i, item := range items {
		las[i] = LearningAreaAttendance(item)
	}

	return AttendanceSection{
		OverallPercentage: overallPct,
		PeriodsTotal:      total,
		PeriodsPresent:    present,
		PeriodsAbsent:     absent,
		PeriodsLate:       late,
		PeriodsExcused:    excused,
		ByLearningArea:    las,
	}
}

func (s *Service) buildAssessmentSection(data *AssessmentData) AssessmentSection {
	if data == nil {
		return AssessmentSection{
			LearningAreas: []LearningAreaAssessment{},
			Sessions:      []AssessmentSessionItem{},
		}
	}
	if data.LearningAreas == nil {
		data.LearningAreas = []LearningAreaAssessment{}
	}
	if data.Sessions == nil {
		data.Sessions = []AssessmentSessionItem{}
	}
	return AssessmentSection{
		LearningAreas: data.LearningAreas,
		Sessions:      data.Sessions,
	}
}

func (s *Service) buildBehaviorSection(notes []BehaviorNoteItem) BehaviorSection {
	if notes == nil {
		notes = []BehaviorNoteItem{}
	}
	return BehaviorSection{
		TotalIncidents: len(notes),
		Notes:          notes,
	}
}
