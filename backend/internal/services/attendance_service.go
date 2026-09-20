package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type AttendanceStudentRecord struct {
	StudentID string `json:"student_id"`
	Status    string `json:"status"`
	Remarks   string `json:"remarks,omitempty"`
}

type CreateAttendanceRequest struct {
	ClassTimetableSlotID string                    `json:"class_timetable_slot_id"`
	AttendanceDate       string                    `json:"attendance_date"`
	Records              []AttendanceStudentRecord `json:"records"`
}

type AttendanceService interface {
	CreateAttendance(ctx context.Context, schoolID uuid.UUID, userID uuid.UUID, req CreateAttendanceRequest) error
	GetAttendanceBySlotAndDate(ctx context.Context, slotID uuid.UUID, date time.Time) ([]sqlc.TimetableAttendance, error)
	GetAttendanceByStudentAndDate(ctx context.Context, studentID uuid.UUID, date time.Time) ([]sqlc.TimetableAttendance, error)
}

type attendanceService struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
	logger  *zap.Logger
}

func NewAttendanceService(pool *pgxpool.Pool, queries *sqlc.Queries) AttendanceService {
	return &attendanceService{queries: queries, pool: pool, logger: zap.L().With(zap.String("service", "attendance"))}
}

func (s *attendanceService) CreateAttendance(ctx context.Context, schoolID uuid.UUID, userID uuid.UUID, req CreateAttendanceRequest) error {
	// Validate required fields
	if req.ClassTimetableSlotID == "" {
		return fmt.Errorf("class_timetable_slot_id is required")
	}
	if req.AttendanceDate == "" {
		return fmt.Errorf("attendance_date is required")
	}
	if len(req.Records) == 0 {
		return fmt.Errorf("at least one attendance record is required")
	}

	// Parse class timetable slot ID
	slotUUID, err := uuid.Parse(req.ClassTimetableSlotID)
	if err != nil {
		return fmt.Errorf("invalid class_timetable_slot_id: %w", err)
	}

	// Parse attendance date
	date, err := time.Parse("2006-01-02", req.AttendanceDate)
	if err != nil {
		return fmt.Errorf("invalid attendance_date format (expected YYYY-MM-DD): %w", err)
	}

	// Parse user ID to get school membership ID for recorded_by_membership_id
	// The userID from locals is the user ID, we need to find the school_memberships record
	// that corresponds to this user in the given school
	membershipRow, err := s.queries.GetSchoolMembershipByUserAndSchool(ctx, sqlc.GetSchoolMembershipByUserAndSchoolParams{
		UserID:   pgtype.UUID{Bytes: userID, Valid: true},
		SchoolID: pgtype.UUID{Bytes: schoolID, Valid: true},
	})
	if err != nil {
		s.logger.Error("failed to get school membership for user", zap.Error(err), zap.String("user_id", userID.String()), zap.String("school_id", schoolID.String()))
		return fmt.Errorf("user not a member of this school: %w", err)
	}

	// Use transaction for batch insert
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			s.logger.Error("rollback failed", zap.Error(rbErr))
		}
	}()

	qtx := s.queries.WithTx(tx)

	for _, record := range req.Records {
		if record.StudentID == "" {
			return fmt.Errorf("student_id is required for all records")
		}
		if record.Status == "" {
			return fmt.Errorf("status is required for all records")
		}

		// Validate status
		validStatuses := map[string]bool{
			"PRESENT": true, "ABSENT": true, "LATE": true, "EXCUSED": true,
		}
		if !validStatuses[record.Status] {
			return fmt.Errorf("invalid status '%s' for student %s: must be PRESENT, ABSENT, LATE, or EXCUSED", record.Status, record.StudentID)
		}

		studentUUID, err := uuid.Parse(record.StudentID)
		if err != nil {
			return fmt.Errorf("invalid student_id '%s': %w", record.StudentID, err)
		}

		remarks := pgtype.Text{}
		if record.Remarks != "" {
			remarks.String = record.Remarks
			remarks.Valid = true
		}

		_, err = qtx.CreateTimetableAttendance(ctx, sqlc.CreateTimetableAttendanceParams{
			SchoolID:               pgtype.UUID{Bytes: schoolID, Valid: true},
			StudentID:              pgtype.UUID{Bytes: studentUUID, Valid: true},
			ClassTimetableSlotID:   pgtype.UUID{Bytes: slotUUID, Valid: true},
			AttendanceDate:         pgtype.Date{Time: date, Valid: true},
			Status:                 sqlc.TimetableAttendanceStatus(record.Status),
			Remarks:                remarks,
			RecordedByMembershipID: pgtype.UUID{Bytes: membershipRow.ID.Bytes, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create attendance for student %s: %w", record.StudentID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit attendance records: %w", err)
	}

	return nil
}

func (s *attendanceService) GetAttendanceBySlotAndDate(ctx context.Context, slotID uuid.UUID, date time.Time) ([]sqlc.TimetableAttendance, error) {
	return s.queries.GetTimetableAttendanceBySlotAndDate(ctx, sqlc.GetTimetableAttendanceBySlotAndDateParams{
		ClassTimetableSlotID: pgtype.UUID{Bytes: slotID, Valid: true},
		AttendanceDate:       pgtype.Date{Time: date, Valid: true},
	})
}

func (s *attendanceService) GetAttendanceByStudentAndDate(ctx context.Context, studentID uuid.UUID, date time.Time) ([]sqlc.TimetableAttendance, error) {
	return s.queries.GetTimetableAttendanceByStudentAndDate(ctx, sqlc.GetTimetableAttendanceByStudentAndDateParams{
		StudentID:      pgtype.UUID{Bytes: studentID, Valid: true},
		AttendanceDate: pgtype.Date{Time: date, Valid: true},
	})
}
