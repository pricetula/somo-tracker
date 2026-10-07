package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type Enrollment struct {
	ID              uuid.UUID `json:"id"`
	StudentID       uuid.UUID `json:"student_id"`
	ClassRoomID     uuid.UUID `json:"class_room_id"`
	AcademicYearID  uuid.UUID `json:"academic_year_id"`
	AcademicTermID  uuid.UUID `json:"academic_term_id"`
	EnrolledAt      time.Time `json:"enrolled_at"`
	Status          string    `json:"status"`
	ClassName       string    `json:"class_name"`
	Grade           string    `json:"grade"`
	Stream          string    `json:"stream"`
	StudentName     string    `json:"student_name"`
	AdmissionNumber string    `json:"admission_number"`
}

type UnassignedStudent struct {
	StudentID       uuid.UUID `json:"student_id"`
	FullName        string    `json:"full_name"`
	AdmissionNumber string    `json:"admission_number"`
}

type CreateEnrollmentRequest struct {
	StudentID      uuid.UUID       `json:"student_id"`
	EnrollmentDate string          `json:"enrollment_date"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type UpdateEnrollmentRequest struct {
	ClassRoomID *uuid.UUID      `json:"class_room_id,omitempty"`
	Status      *string         `json:"status,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type ListEnrollmentsParams struct {
	SchoolID     uuid.UUID
	Page         int
	Limit        int
	ClassRoomID  *uuid.UUID
	StudentID    *uuid.UUID
	StatusFilter string
}

type EnrollmentsService interface {
	CreateEnrollments(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, reqs []CreateEnrollmentRequest) ([]*Enrollment, error)
	ListEnrollments(ctx context.Context, params ListEnrollmentsParams) ([]*Enrollment, int, error)
	ListUnassignedStudents(ctx context.Context, schoolID uuid.UUID, academicYearID uuid.UUID, page, limit int) ([]*UnassignedStudent, int, error)
	ListUnassignedStudentsCurrentYear(ctx context.Context, schoolID uuid.UUID, page, limit int) ([]*UnassignedStudent, int, error)
	ListUnassignedStudentsByClass(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, page, limit int) ([]*UnassignedStudent, int, error)
	UpdateEnrollment(ctx context.Context, id uuid.UUID, req UpdateEnrollmentRequest) (*Enrollment, error)
	DeleteEnrollment(ctx context.Context, id uuid.UUID) error
	GetActiveEnrollmentByStudent(ctx context.Context, studentID, academicYearID uuid.UUID) (*Enrollment, error)
}

type enrollmentsService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewEnrollmentsService(queries *sqlc.Queries, logger *zap.Logger) EnrollmentsService {
	return &enrollmentsService{
		queries: queries,
		logger:  logger.With(zap.String("service", "enrollments")),
	}
}

func uuidToPG(u uuid.UUID) pgtype.UUID {
	var b [16]byte
	copy(b[:], u[:])
	return pgtype.UUID{Bytes: b, Valid: true}
}
func pgToUUID(p pgtype.UUID) uuid.UUID { var u uuid.UUID; copy(u[:], p.Bytes[:]); return u }

func (s *enrollmentsService) CreateEnrollments(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, reqs []CreateEnrollmentRequest) ([]*Enrollment, error) {
	if len(reqs) == 0 {
		return nil, fmt.Errorf("no enrollment requests provided")
	}
	classRow, err := s.queries.GetClassRoom(ctx, uuidToPG(classRoomID))
	if err != nil {
		s.logger.Error("get class room failed", zap.Error(err), zap.String("class_room_id", classRoomID.String()))
		return nil, fmt.Errorf("failed to resolve class room: %w", err)
	}
	if !classRow.SchoolID.Valid || classRow.SchoolID.Bytes != schoolID {
		return nil, fmt.Errorf("class room does not belong to school")
	}
	academicYearID := pgToUUID(classRow.AcademicYearID)
	academicTermRow, termErr := s.queries.GetCurrentAcademicTermBySchool(ctx, uuidToPG(schoolID))
	if termErr != nil {
		s.logger.Error("get current academic term failed", zap.Error(termErr), zap.String("school_id", schoolID.String()))
		return nil, fmt.Errorf("failed to resolve academic term: %w", termErr)
	}
	var academicTermID pgtype.UUID
	if academicTermRow.ID.Valid {
		academicTermID = academicTermRow.ID
	}
	var results []*Enrollment
	for _, req := range reqs {
		if req.StudentID == uuid.Nil {
			return nil, fmt.Errorf("student_id is required")
		}
		enrolledAt, err := time.Parse("2006-01-02", req.EnrollmentDate)
		if err != nil {
			return nil, fmt.Errorf("invalid enrollment_date: %w", err)
		}
		metadata := json.RawMessage("{}")
		if len(req.Metadata) > 0 {
			metadata = req.Metadata
		}
		enrolledAtPG := pgtype.Timestamptz{Time: enrolledAt, Valid: true}
		row, err := s.queries.CreateEnrollment(ctx, sqlc.CreateEnrollmentParams{
			SchoolID:       uuidToPG(schoolID),
			StudentID:      uuidToPG(req.StudentID),
			ClassRoomID:    uuidToPG(classRoomID),
			AcademicYearID: uuidToPG(academicYearID),
			AcademicTermID: academicTermID,
			EnrolledAt:     enrolledAtPG,
			Metadata:       metadata,
		})
		if err != nil {
			s.logger.Error("create enrollment failed", zap.Error(err), zap.String("student_id", req.StudentID.String()))
			return nil, fmt.Errorf("failed to create enrollment for student %s: %w", req.StudentID, err)
		}
		results = append(results, mapEnrollmentRow(row))
	}
	return results, nil
}

func (s *enrollmentsService) ListEnrollments(ctx context.Context, params ListEnrollmentsParams) ([]*Enrollment, int, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var classRoomIDPG, studentIDPG pgtype.UUID
	if params.ClassRoomID != nil && *params.ClassRoomID != uuid.Nil {
		classRoomIDPG = uuidToPG(*params.ClassRoomID)
	}
	if params.StudentID != nil && *params.StudentID != uuid.Nil {
		studentIDPG = uuidToPG(*params.StudentID)
	}

	statusFilter := params.StatusFilter

	rows, err := s.queries.ListEnrollments(ctx, sqlc.ListEnrollmentsParams{
		SchoolID: uuidToPG(params.SchoolID),
		Column2:  classRoomIDPG,
		Column3:  studentIDPG,
		Column4:  statusFilter,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		s.logger.Error("list enrollments failed", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list enrollments: %w", err)
	}

	totalRow, err := s.queries.CountEnrollments(ctx, sqlc.CountEnrollmentsParams{
		SchoolID: uuidToPG(params.SchoolID),
		Column2:  classRoomIDPG,
		Column3:  studentIDPG,
		Column4:  params.StatusFilter,
	})
	if err != nil {
		s.logger.Error("count enrollments failed", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count enrollments: %w", err)
	}

	var enrollments []*Enrollment
	for _, r := range rows {
		e := &Enrollment{
			ID:              pgToUUID(r.ID),
			StudentID:       pgToUUID(r.StudentID),
			ClassRoomID:     pgToUUID(r.ClassRoomID),
			AcademicYearID:  pgToUUID(r.AcademicYearID),
			AcademicTermID:  pgToUUID(r.AcademicTermID),
			EnrolledAt:      r.EnrolledAt.Time,
			Status:          string(r.Status),
			ClassName:       r.ClassName,
			Grade:           r.Grade,
			Stream:          r.Stream.String,
			StudentName:     r.StudentName,
			AdmissionNumber: r.AdmissionNumber,
		}
		enrollments = append(enrollments, e)
	}
	return enrollments, int(totalRow), nil
}

func (s *enrollmentsService) ListUnassignedStudents(ctx context.Context, schoolID uuid.UUID, academicYearID uuid.UUID, page, limit int) ([]*UnassignedStudent, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	rows, err := s.queries.ListUnassignedStudents(ctx, sqlc.ListUnassignedStudentsParams{
		SchoolID:       uuidToPG(schoolID),
		AcademicYearID: uuidToPG(academicYearID),
		Limit:          int32(limit),
		Offset:         int32(offset),
	})
	if err != nil {
		s.logger.Error("list unassigned students failed", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list unassigned students: %w", err)
	}
	count, err := s.queries.CountUnassignedStudents(ctx, sqlc.CountUnassignedStudentsParams{
		SchoolID:       uuidToPG(schoolID),
		AcademicYearID: uuidToPG(academicYearID),
	})
	if err != nil {
		s.logger.Error("count unassigned students failed", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count unassigned students: %w", err)
	}
	var students []*UnassignedStudent
	for _, r := range rows {
		students = append(students, &UnassignedStudent{
			StudentID:       pgToUUID(r.StudentID),
			FullName:        r.FullName,
			AdmissionNumber: r.AdmissionNumber,
		})
	}
	return students, int(count), nil
}

func (s *enrollmentsService) ListUnassignedStudentsByClass(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, page, limit int) ([]*UnassignedStudent, int, error) {
	classRow, err := s.queries.GetClassRoom(ctx, uuidToPG(classRoomID))
	if err != nil {
		s.logger.Error("get class room failed", zap.Error(err), zap.String("class_room_id", classRoomID.String()))
		return nil, 0, fmt.Errorf("failed to resolve class room: %w", err)
	}
	if !classRow.SchoolID.Valid || classRow.SchoolID.Bytes != schoolID {
		return nil, 0, fmt.Errorf("class room does not belong to school")
	}
	academicYearID := pgToUUID(classRow.AcademicYearID)
	return s.ListUnassignedStudents(ctx, schoolID, academicYearID, page, limit)
}

func (s *enrollmentsService) ListUnassignedStudentsCurrentYear(ctx context.Context, schoolID uuid.UUID, page, limit int) ([]*UnassignedStudent, int, error) {
	academicYearIDPg, err := s.queries.GetCurrentAcademicYearBySchool(ctx, uuidToPG(schoolID))
	if err != nil {
		s.logger.Error("get current academic year failed", zap.Error(err), zap.String("school_id", schoolID.String()))
		return nil, 0, fmt.Errorf("failed to resolve academic year: %w", err)
	}
	if !academicYearIDPg.Valid {
		return nil, 0, fmt.Errorf("academic year not found")
	}
	academicYearID := pgToUUID(academicYearIDPg)
	return s.ListUnassignedStudents(ctx, schoolID, academicYearID, page, limit)
}

func (s *enrollmentsService) UpdateEnrollment(ctx context.Context, id uuid.UUID, req UpdateEnrollmentRequest) (*Enrollment, error) {
	metadata := json.RawMessage("{}")
	if len(req.Metadata) > 0 {
		metadata = req.Metadata
	}
	classRoomID := pgtype.UUID{Valid: false}
	if req.ClassRoomID != nil {
		classRoomID = uuidToPG(*req.ClassRoomID)
	}
	var status sqlc.EnrollmentStatus
	if req.Status != nil {
		status = sqlc.EnrollmentStatus(*req.Status)
	} else {
		status = ""
	}
	row, err := s.queries.UpdateEnrollment(ctx, sqlc.UpdateEnrollmentParams{
		ID:          uuidToPG(id),
		ClassRoomID: classRoomID,
		Status:      status,
		Metadata:    metadata,
	})
	if err != nil {
		s.logger.Error("update enrollment failed", zap.Error(err), zap.String("id", id.String()))
		return nil, fmt.Errorf("failed to update enrollment: %w", err)
	}
	return mapEnrollmentRow(row), nil
}

func (s *enrollmentsService) DeleteEnrollment(ctx context.Context, id uuid.UUID) error {
	err := s.queries.DeleteEnrollment(ctx, uuidToPG(id))
	if err != nil {
		s.logger.Error("delete enrollment failed", zap.Error(err), zap.String("id", id.String()))
		return fmt.Errorf("failed to delete enrollment: %w", err)
	}
	return nil
}

func (s *enrollmentsService) GetActiveEnrollmentByStudent(ctx context.Context, studentID, academicYearID uuid.UUID) (*Enrollment, error) {
	row, err := s.queries.GetActiveEnrollmentByStudent(ctx, sqlc.GetActiveEnrollmentByStudentParams{
		StudentID:      uuidToPG(studentID),
		AcademicYearID: uuidToPG(academicYearID),
	})
	if err != nil {
		s.logger.Error("get active enrollment failed", zap.Error(err), zap.String("student_id", studentID.String()))
		return nil, fmt.Errorf("failed to get active enrollment: %w", err)
	}
	return mapEnrollmentRow(row), nil
}

func mapEnrollmentRow(r sqlc.StudentClassEnrollment) *Enrollment {
	return &Enrollment{
		ID:             pgToUUID(r.ID),
		StudentID:      pgToUUID(r.StudentID),
		ClassRoomID:    pgToUUID(r.ClassRoomID),
		AcademicYearID: pgToUUID(r.AcademicYearID),
		AcademicTermID: pgToUUID(r.AcademicTermID),
		EnrolledAt:     r.EnrolledAt.Time,
		Status:         string(r.Status),
	}
}
