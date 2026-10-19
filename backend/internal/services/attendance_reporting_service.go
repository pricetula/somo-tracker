package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"somotracker/backend/internal/database/sqlc"
)

type AttendanceStatsItem struct {
	StudentID       uuid.UUID `json:"student_id"`
	FullName        string    `json:"full_name,omitempty"`
	AdmissionNumber string    `json:"admission_number,omitempty"`
	PresentCount    int32     `json:"present_count"`
	AbsentCount     int32     `json:"absent_count"`
	LateCount       int32     `json:"late_count"`
	AttendanceRate  float64   `json:"attendance_rate"`
}

type AttendanceReportingService struct {
	queries *sqlc.Queries
}

func NewAttendanceReportingService(queries *sqlc.Queries) *AttendanceReportingService {
	return &AttendanceReportingService{queries: queries}
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil || !f8.Valid {
		return 0
	}
	return f8.Float64
}

func (s *AttendanceReportingService) ListStudentTermStats(ctx context.Context, schoolID uuid.UUID, termID uuid.UUID, page, limit int) ([]AttendanceStatsItem, int, error) {
	offset := (page - 1) * limit
	rows, err := s.queries.ListStudentTermAttendanceStats(ctx, sqlc.ListStudentTermAttendanceStatsParams{
		SchoolID:       uuidToPG(schoolID),
		AcademicTermID: uuidToPG(termID),
		Limit:          int32(limit),
		Offset:         int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]AttendanceStatsItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, AttendanceStatsItem{
			StudentID:      pgToUUID(r.StudentID),
			PresentCount:   r.PresentCount,
			AbsentCount:    r.AbsentCount,
			LateCount:      r.LateCount,
			AttendanceRate: numericToFloat64(r.AttendanceRate),
		})
	}
	return items, len(items), nil
}
