package students

import (
	"context"

	"somotracker/backend/internal/attendance"
	"somotracker/backend/internal/behavior"
)

// ─── Cross-domain handler providers ───────────────────────────────────────
//
// The students handler needs behavior notes and attendance summaries for the
// student detail page. The provider function types live in handler.go so the
// handler itself stays decoupled; this file is the joint point that builds
// them from the behavior and attendance services. Neither behavior nor
// attendance imports students, so the dependency remains one-directional
// (zero circular imports).

// wireHandlerProviders injects the behavior + attendance adapters into the
// students handler. Called from the fx module via fx.Invoke.
func wireHandlerProviders(
	h *Handler,
	behSvc *behavior.Service,
	attSvc *attendance.Service,
) {
	h.SetBehaviorNotesProvider(func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error) {
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
				Status:       string(n.Status),
				IsUrgent:     n.IsUrgent,
			})
		}
		return items, nil
	})

	h.SetAttendanceProvider(func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error) {
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
	})
}
