package reports

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/assessments"
	"somotracker/backend/internal/attendance"
	"somotracker/backend/internal/behavior"
	"somotracker/backend/internal/cbcclasses"
	"somotracker/backend/internal/cbcschools"
	"somotracker/backend/internal/students"
)

// ─── Cross-domain provider adapters ───────────────────────────────────────
//
// The reports module is the application-layer orchestrator for term report
// cards: it declares its data sources as function types (StudentProvider,
// TermProvider, …) in domain.go and builds them here from the owning domain
// services. This package is the joint point that may import the domains it
// orchestrates — the domains themselves never import reports (zero circular
// imports).

// newStudentProvider maps students.Service.GetDetail (+ cbcclasses class
// metadata for grade level / stream) into the StudentProvider shape.
func newStudentProvider(
	studentSvc *students.Service,
	classSvc *cbcclasses.Service,
	log *zap.SugaredLogger,
) StudentProvider {
	return func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error) {
		detail, err := studentSvc.GetDetail(ctx, studentID, tenantID, schoolID)
		if err != nil {
			return nil, err
		}

		out := &TermReportStudent{
			ID:              detail.ID,
			FullName:        detail.FullName,
			Gender:          detail.Gender,
			AdmissionNumber: detail.AdmissionNumber,
			UPINumber:       detail.UPINumber,
		}
		if detail.ClassName != nil {
			out.ClassName = *detail.ClassName
		}
		if detail.ClassID != nil && *detail.ClassID != "" {
			cls, clsErr := classSvc.GetClass(ctx, *detail.ClassID, tenantID, schoolID)
			if clsErr != nil {
				// Best-effort enrichment — the report still works without
				// grade level / stream. Log once here (adapter is the
				// handling layer) and continue.
				log.Warnw("reports student provider: class lookup failed for report",
					"student_id", studentID,
					"class_id", *detail.ClassID,
					"error", clsErr.Error(),
				)
			} else {
				out.GradeLevel = cls.GradeLevel
				out.StreamName = cls.StreamName
			}
		}
		return out, nil
	}
}

// newTermProvider resolves a term's name/number plus its academic year and
// school names from the academicyears and cbcschools domains.
func newTermProvider(
	aySvc *academicyears.Service,
	schoolSvc *cbcschools.Service,
	log *zap.SugaredLogger,
) TermProvider {
	return func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error) {
		terms, err := aySvc.ListTerms(ctx, tenantID, schoolID, nil)
		if err != nil {
			return nil, err
		}
		var term *academicyears.AcademicTerm
		for i := range terms {
			if terms[i].ID == termID {
				term = &terms[i]
				break
			}
		}
		if term == nil {
			return nil, fmt.Errorf("reports.termReportProvider: term %s not found: %w", termID, academicyears.ErrNotFound)
		}

		info := &TermInfo{
			TermID:     term.ID,
			TermName:   term.Name,
			TermNumber: term.TermNumber,
		}

		// Academic year name (best-effort)
		if years, err := aySvc.ListYears(ctx, tenantID, schoolID); err == nil {
			for i := range years {
				if years[i].ID == term.AcademicYearID {
					info.AcademicYear = years[i].Name
					break
				}
			}
		} else {
			log.Warnw("reports term provider: academic year lookup failed",
				"term_id", termID, "error", err.Error())
		}

		// School name (best-effort)
		if school, err := schoolSvc.GetSchool(ctx, schoolID, tenantID); err == nil {
			info.SchoolName = school.Name
		} else {
			log.Warnw("reports term provider: school lookup failed",
				"school_id", schoolID, "error", err.Error())
		}

		return info, nil
	}
}

// newAttendanceProvider maps attendance term summaries into the reports shape.
func newAttendanceProvider(attSvc *attendance.Service) AttendanceProvider {
	return func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error) {
		resp, err := attSvc.GetStudentTermSummary(ctx, tenantID, schoolID, studentID, termID)
		if err != nil {
			return nil, err
		}
		items := make([]AttendanceSummaryItem, 0, len(resp.Items))
		for _, it := range resp.Items {
			items = append(items, AttendanceSummaryItem{
				LearningAreaID:   it.LearningAreaID,
				LearningAreaName: it.LearningAreaName,
				PeriodsTotal:     it.PeriodsTotal,
				PeriodsPresent:   it.PeriodsPresent,
				PeriodsAbsent:    it.PeriodsAbsent,
				PeriodsLate:      it.PeriodsLate,
				PeriodsExcused:   it.PeriodsExcused,
				Percentage:       it.AttendancePercentage,
			})
		}
		return items, nil
	}
}

// newAssessmentProvider compiles the report's assessment section from the
// "Last One" term grades plus the published parent-view assessment sessions.
func newAssessmentProvider(asmtSvc *assessments.Service) AssessmentProvider {
	return func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*AssessmentData, error) {
		grades, err := asmtSvc.GetStudentTermGrades(ctx, tenantID, schoolID, studentID, termID)
		if err != nil {
			return nil, err
		}
		sessions, err := asmtSvc.GetParentAssessments(ctx, tenantID, schoolID, studentID, termID)
		if err != nil {
			return nil, err
		}

		data := &AssessmentData{
			LearningAreas: make([]LearningAreaAssessment, 0, len(grades)),
			Sessions:      make([]AssessmentSessionItem, 0, len(sessions)),
		}
		for _, g := range grades {
			data.LearningAreas = append(data.LearningAreas, LearningAreaAssessment{
				LearningAreaID:   g.LearningAreaID,
				LearningAreaName: g.LearningAreaName,
				FinalLevel:       g.FinalLevel,
				AssessmentCount:  g.AssessmentCount,
			})
		}
		for _, s := range sessions {
			data.Sessions = append(data.Sessions, AssessmentSessionItem{
				SessionID:        s.SessionID,
				SessionName:      s.SessionName,
				EvaluationMethod: s.EvaluationMethod,
				ScheduledDate:    s.ScheduledDate,
				RawScore:         s.RawScore,
				MaxPoints:        s.MaxPoints,
				PerformanceLevel: s.PerformanceLevel,
			})
		}
		return data, nil
	}
}

// newBehaviorProvider maps approved behavior notes into the reports shape.
func newBehaviorProvider(behSvc *behavior.Service) BehaviorProvider {
	return func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error) {
		notes, err := behSvc.GetNotesByStudentTerm(ctx, tenantID, schoolID, studentID, termID)
		if err != nil {
			return nil, err
		}
		items := make([]BehaviorNoteItem, 0, len(notes))
		for _, n := range notes {
			items = append(items, BehaviorNoteItem{
				ID:           n.ID,
				CategoryName: n.CategoryName,
				Description:  n.Description,
				Date:         n.Date.Format("2006-01-02"),
				IsUrgent:     n.IsUrgent,
			})
		}
		return items, nil
	}
}
