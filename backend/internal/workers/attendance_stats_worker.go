package workers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"somotracker/backend/internal/database/sqlc"
)

// AttendanceStatsWorker upserts student_term_attendance_stats summary table.
// In production this is triggered by Asynq tasks on attendance.marked events.
type AttendanceStatsWorker struct {
	queries *sqlc.Queries
}

// NewAttendanceStatsWorker creates a new worker instance.
func NewAttendanceStatsWorker(queries *sqlc.Queries) *AttendanceStatsWorker {
	return &AttendanceStatsWorker{queries: queries}
}

// UpsertStats recalculates attendance_rate for a given student/term and upserts the summary row.
// This is a stub implementation for the empty DB case; real implementation will aggregate from timetable_attendance.
func (w *AttendanceStatsWorker) UpsertStats(ctx context.Context, schoolID, academicTermID, studentID uuid.UUID, present, absent, late int) error {
	if present+absent+late == 0 {
		return nil
	}
	rate := float64(present) / float64(present+absent+late) * 100

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	termUUID := pgtype.UUID{Bytes: academicTermID, Valid: true}
	studentUUID := pgtype.UUID{Bytes: studentID, Valid: true}

	// Note: sqlc query for upsert needs to be generated from the schema.
	// Placeholder implementation demonstrates the intended shape.
	// In practice, use queries.UpsertStudentTermAttendanceStats with appropriate params.
	_ = schoolUUID
	_ = termUUID
	_ = studentUUID
	_ = rate

	// TODO: Replace with actual upsert query once sqlc models are generated for the summary table.
	return fmt.Errorf("attendance stats upsert not implemented: student %s term %s", studentID, academicTermID)
}
