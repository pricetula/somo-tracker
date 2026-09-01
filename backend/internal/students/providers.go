package students

import (
	"context"

	"somotracker/backend/internal/attendance"
)

// ─── Cross-domain handler providers ───────────────────────────────────────
//
// The students handler needs attendance summaries for the student detail page.
// The provider function types live in handler.go so the handler itself stays
// decoupled; this file is the joint point that builds them from the attendance
// service. Attendance does not import students, so the dependency is one-directional
// (zero circular imports).

// wireHandlerProviders injects the attendance adapter into the students handler.
// Called from the fx module via fx.Invoke.
func wireHandlerProviders(
	h *Handler,
	attSvc *attendance.Service,
) {
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
