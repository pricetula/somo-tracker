package services

import (
	"context"
	"fmt"
	"strings"
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

type AttendanceStatus string

const (
	AttendanceStatusSubmitted  AttendanceStatus = "SUBMITTED"
	AttendanceStatusInProgress AttendanceStatus = "IN_PROGRESS"
	AttendanceStatusMissed     AttendanceStatus = "MISSED"
)

type CreateAttendanceRequest struct {
	ClassTimetableSlotID string                    `json:"class_timetable_slot_id"`
	AttendanceDate       string                    `json:"attendance_date"`
	Records              []AttendanceStudentRecord `json:"records"`
}

type AttendanceSession struct {
	SlotID           uuid.UUID `json:"slot_id"`
	ClassID          uuid.UUID `json:"class_id"`
	ClassName        string    `json:"class_name"`
	Grade            string    `json:"grade"`
	Stream           string    `json:"stream"`
	Subject          string    `json:"subject"`
	TeacherName      string    `json:"teacher_name"`
	TeacherID        uuid.UUID `json:"teacher_id"`
	DayOfWeek        int       `json:"day_of_week"`
	TimeSlotName     string    `json:"time_slot_name"`
	StartTime        string    `json:"start_time"`
	EndTime          string    `json:"end_time"`
	AttendanceDate   string    `json:"attendance_date"`
	Status           string    `json:"status"`
	TotalStudents    int       `json:"total_students"`
	RecordedStudents int       `json:"recorded_students"`
}

type ListAttendanceSessionsParams struct {
	SchoolID      uuid.UUID
	Page          int
	Limit         int
	Search        string
	StatusFilter  string
	DateFrom      *time.Time
	DateTo        *time.Time
	GradeFilter   []string
	StreamFilter  []string
	SubjectFilter []string
	TeacherFilter []string
}

type AttendanceService interface {
	CreateAttendance(ctx context.Context, schoolID uuid.UUID, userID uuid.UUID, req CreateAttendanceRequest) error
	GetAttendanceBySlotAndDate(ctx context.Context, slotID uuid.UUID, date time.Time) ([]sqlc.TimetableAttendance, error)
	GetAttendanceByStudentAndDate(ctx context.Context, studentID uuid.UUID, date time.Time) ([]sqlc.TimetableAttendance, error)
	ListAttendanceSessions(ctx context.Context, params ListAttendanceSessionsParams) ([]AttendanceSession, int, error)
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
	if req.ClassTimetableSlotID == "" {
		return fmt.Errorf("class_timetable_slot_id is required")
	}
	if req.AttendanceDate == "" {
		return fmt.Errorf("attendance_date is required")
	}
	if len(req.Records) == 0 {
		return fmt.Errorf("at least one attendance record is required")
	}

	slotUUID, err := uuid.Parse(req.ClassTimetableSlotID)
	if err != nil {
		return fmt.Errorf("invalid class_timetable_slot_id: %w", err)
	}

	date, err := time.Parse("2006-01-02", req.AttendanceDate)
	if err != nil {
		return fmt.Errorf("invalid attendance_date format (expected YYYY-MM-DD): %w", err)
	}

	membershipRow, err := s.queries.GetSchoolMembershipByUserAndSchool(ctx, sqlc.GetSchoolMembershipByUserAndSchoolParams{
		UserID:   pgtype.UUID{Bytes: userID, Valid: true},
		SchoolID: pgtype.UUID{Bytes: schoolID, Valid: true},
	})
	if err != nil {
		s.logger.Error("failed to get school membership for user", zap.Error(err), zap.String("user_id", userID.String()), zap.String("school_id", schoolID.String()))
		return fmt.Errorf("user not a member of this school: %w", err)
	}

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

func (s *attendanceService) ListAttendanceSessions(ctx context.Context, params ListAttendanceSessionsParams) ([]AttendanceSession, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 200 {
		params.Limit = 50
	}
	if params.DateFrom == nil {
		var ayStart time.Time
		if err := s.pool.QueryRow(ctx, `SELECT start_date FROM academic_years WHERE school_id = $1 ORDER BY ay.start_date DESC LIMIT 1`, params.SchoolID).Scan(&ayStart); err == nil {
			params.DateFrom = &ayStart
		} else {
			fallback := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -30)
			params.DateFrom = &fallback
		}
	}
	if params.DateTo == nil {
		now := time.Now().Truncate(24 * time.Hour)
		params.DateTo = &now
	}

	var termStart, termEnd time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT at.start_date, at.end_date FROM academic_terms at JOIN academic_years ay ON ay.id = at.academic_year_id WHERE ay.school_id = $1 AND at.start_date <= $2 AND at.end_date >= $3
		ORDER BY at.start_date DESC LIMIT 1
	`, params.SchoolID, *params.DateTo, *params.DateFrom).Scan(&termStart, &termEnd)
	if err != nil {
		s.logger.Warn("no academic term found for date range, using provided dates", zap.Error(err))
		termStart = *params.DateFrom
		termEnd = *params.DateTo
	}

	if params.DateFrom.Before(termStart) {
		*params.DateFrom = termStart
	}
	if params.DateTo.After(termEnd) {
		*params.DateTo = termEnd
	}

	s.logger.Debug("ListAttendanceSessions params",
		zap.String("school_id", params.SchoolID.String()),
		zap.String("date_from", params.DateFrom.Format("2006-01-02")),
		zap.String("date_to", params.DateTo.Format("2006-01-02")),
		zap.String("term_start", termStart.Format("2006-01-02")),
		zap.String("term_end", termEnd.Format("2006-01-02")),
	)

	baseQuery := `
		WITH slots AS (
			SELECT 
				cts.id AS slot_id,
				cts.class_room_id,
				cts.day_of_week,
				cts.subject_id,
				cts.teacher_membership_id,
				cts.time_slot_id,
				cr.name AS class_name,
				cr.id AS class_id,
				gl.local_label AS grade,
				COALESCE(cr.stream, '') AS stream,
				s.name AS subject_name,
				u.full_name AS teacher_name,
				sm.user_id AS teacher_user_id,
				ts.name AS time_slot_name,
				ts.start_time,
				ts.end_time
			FROM class_timetable_slots cts
			JOIN class_rooms cr ON cr.id = cts.class_room_id
			JOIN grade_levels gl ON gl.id = cr.grade_level_id
			JOIN subjects s ON s.id = cts.subject_id
			JOIN school_memberships sm ON sm.id = cts.teacher_membership_id
			JOIN users u ON u.id = sm.user_id
			JOIN time_slots ts ON ts.id = cts.time_slot_id
			WHERE cts.school_id = $1
			  AND cts.academic_term_id = (SELECT at.id FROM academic_terms at JOIN academic_years ay ON ay.id = at.academic_year_id WHERE ay.school_id = $1 AND at.start_date <= $2 AND at.end_date >= $3 ORDER BY at.start_date DESC LIMIT 1)
		),
		dates AS (
			SELECT generate_series($4, $5, '1 day'::interval)::date AS attendance_date
		),
		slot_dates AS (
			SELECT s.*, d.attendance_date
			FROM slots s
			JOIN dates d ON EXTRACT(DOW FROM d.attendance_date)::int = s.day_of_week
			WHERE d.attendance_date BETWEEN $4 AND $5
		),
		attendance_counts AS (
			SELECT 
				sd.slot_id,
				sd.attendance_date,
				COUNT(ta.id) AS recorded_students
			FROM slot_dates sd
			LEFT JOIN timetable_attendance ta
			  ON ta.class_timetable_slot_id = sd.slot_id
			 AND ta.attendance_date = sd.attendance_date
			GROUP BY sd.slot_id, sd.attendance_date
		),
		student_counts AS (
			SELECT 
				cr.id AS class_id,
				COUNT(sce.id) AS total_students
			FROM class_rooms cr
			LEFT JOIN student_class_enrollments sce
			  ON sce.class_room_id = cr.id
			 AND sce.status = 'ACTIVE'
			WHERE cr.school_id = $1
			  AND cr.academic_year_id = (SELECT ay.id FROM academic_years ay WHERE ay.school_id = $1 ORDER BY ay.start_date DESC LIMIT 1)
			GROUP BY cr.id
		)
	SELECT 
		sd.slot_id,
		sd.class_id,
		sd.class_name,
		sd.grade,
		sd.stream,
		sd.subject_name,
		sd.teacher_name,
		sd.teacher_user_id,
		sd.day_of_week,
		sd.time_slot_name,
		sd.start_time,
		sd.end_time,
		sd.attendance_date,
		COALESCE(ac.recorded_students, 0) AS recorded_students,
		COALESCE(sc.total_students, 0) AS total_students
	FROM slot_dates sd
	LEFT JOIN attendance_counts ac
	  ON ac.slot_id = sd.slot_id AND ac.attendance_date = sd.attendance_date
	LEFT JOIN student_counts sc ON sc.class_id = sd.class_id
	WHERE 1=1
`

	args := []interface{}{params.SchoolID, *params.DateTo, *params.DateFrom, *params.DateFrom, *params.DateTo}
	paramIdx := 6

	if strings.TrimSpace(params.Search) != "" {
		baseQuery += fmt.Sprintf(" AND (sd.class_name ILIKE $%d OR sd.teacher_name ILIKE $%d OR sd.subject_name ILIKE $%d)", paramIdx, paramIdx, paramIdx)
		args = append(args, "%"+params.Search+"%")
		paramIdx++
	}
	if len(params.GradeFilter) > 0 {
		placeholders := make([]string, len(params.GradeFilter))
		for i := range params.GradeFilter {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args = append(args, params.GradeFilter[i])
		}
		baseQuery += fmt.Sprintf(" AND sd.grade IN (%s)", strings.Join(placeholders, ","))
		paramIdx += len(params.GradeFilter)
	}
	if len(params.StreamFilter) > 0 {
		placeholders := make([]string, len(params.StreamFilter))
		for i := range params.StreamFilter {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args = append(args, params.StreamFilter[i])
		}
		baseQuery += fmt.Sprintf(" AND sd.stream IN (%s)", strings.Join(placeholders, ","))
		paramIdx += len(params.StreamFilter)
	}
	if len(params.SubjectFilter) > 0 {
		placeholders := make([]string, len(params.SubjectFilter))
		for i := range params.SubjectFilter {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args = append(args, params.SubjectFilter[i])
		}
		baseQuery += fmt.Sprintf(" AND sd.subject_name IN (%s)", strings.Join(placeholders, ","))
		paramIdx += len(params.SubjectFilter)
	}
	if len(params.TeacherFilter) > 0 {
		placeholders := make([]string, len(params.TeacherFilter))
		for i := range params.TeacherFilter {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args = append(args, params.TeacherFilter[i])
		}
		baseQuery += fmt.Sprintf(" AND sd.teacher_name IN (%s)", strings.Join(placeholders, ","))
		paramIdx += len(params.TeacherFilter)
	}

	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + ") AS cnt"
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		s.logger.Error("list attendance sessions count failed", zap.Error(err), zap.String("query", countQuery), zap.Any("args", args))
		return nil, 0, fmt.Errorf("internal_error: failed to count attendance sessions")
	}

	baseQuery += fmt.Sprintf(" ORDER BY sd.attendance_date DESC, sd.start_time LIMIT $%d OFFSET $%d", paramIdx, paramIdx+1)
	args = append(args, params.Limit, (params.Page-1)*params.Limit)

	rows, err := s.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		s.logger.Error("list attendance sessions query failed", zap.Error(err))
		return nil, 0, fmt.Errorf("internal_error: failed to list attendance sessions")
	}
	defer rows.Close()

	var sessions []AttendanceSession
	for rows.Next() {
		var sess AttendanceSession
		var slotID, classID, teacherUserID uuid.UUID
		var startTime, endTime pgtype.Time
		var attendanceDate pgtype.Date
		var recordedStudents, totalStudents int

		if err := rows.Scan(
			&slotID, &classID, &sess.ClassName, &sess.Grade, &sess.Stream,
			&sess.Subject, &sess.TeacherName, &teacherUserID,
			&sess.DayOfWeek, &sess.TimeSlotName, &startTime, &endTime, &attendanceDate,
			&recordedStudents, &totalStudents,
		); err != nil {
			s.logger.Error("list attendance sessions scan failed", zap.Error(err))
			continue
		}

		sess.SlotID = slotID
		sess.ClassID = classID
		sess.TeacherID = teacherUserID
		sess.RecordedStudents = recordedStudents
		sess.TotalStudents = totalStudents

		if attendanceDate.Valid {
			sess.AttendanceDate = attendanceDate.Time.Format("2006-01-02")
		}
		if startTime.Valid {
			sess.StartTime = formatTime(startTime)
		}
		if endTime.Valid {
			sess.EndTime = formatTime(endTime)
		}

		if totalStudents == 0 {
			sess.Status = "NO_STUDENTS"
		} else if recordedStudents >= totalStudents {
			sess.Status = string(AttendanceStatusSubmitted)
		} else if recordedStudents > 0 {
			sess.Status = string(AttendanceStatusInProgress)
		} else {
			attDate, _ := time.Parse("2006-01-02", sess.AttendanceDate)
			if attDate.Before(time.Now().Truncate(24 * time.Hour)) {
				sess.Status = string(AttendanceStatusMissed)
			} else {
				sess.Status = string(AttendanceStatusInProgress)
			}
		}

		sessions = append(sessions, sess)
	}

	return sessions, total, nil
}

func formatTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	hours := t.Microseconds / 3600000000
	minutes := (t.Microseconds % 3600000000) / 60000000
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}
