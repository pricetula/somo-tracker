package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"somotracker/backend/internal/database/sqlc"
)

type StudentListItem struct {
	StudentID       string `json:"student_id"`
	AdmissionNumber string `json:"admission_number"`
	FullName        string `json:"full_name"`
	DateOfBirth     string `json:"date_of_birth"`
	Gender          string `json:"gender"`
	ClassID         string `json:"class_id"`
	ClassName       string `json:"class_name"`
}

type StudentListResponse struct {
	Items []StudentListItem `json:"items"`
	Total int               `json:"total"`
}

type StudentsService interface {
	ListStudents(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, classID *uuid.UUID) (*StudentListResponse, error)
	DeleteStudents(ctx context.Context, schoolID uuid.UUID, studentIDs []uuid.UUID) error
}

type studentsService struct {
	queries *sqlc.Queries
}

func NewStudentsService(queries *sqlc.Queries) StudentsService {
	return &studentsService{queries: queries}
}

func (s *studentsService) ListStudents(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, classID *uuid.UUID) (*StudentListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	var classUUID pgtype.UUID
	if classID != nil {
		classUUID = pgtype.UUID{Bytes: *classID, Valid: true}
	}
	rows, err := s.queries.ListStudents(ctx, sqlc.ListStudentsParams{
		SchoolID: schoolUUID,
		Column2:  search,
		Column5:  classUUID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}

	total, err := s.queries.CountStudents(ctx, sqlc.CountStudentsParams{
		SchoolID: schoolUUID,
		Column2:  search,
		Column3:  classUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("count students: %w", err)
	}

	items := make([]StudentListItem, 0, len(rows))
	for _, r := range rows {
		dobStr := ""
		if r.DateOfBirth.Valid {
			dobStr = r.DateOfBirth.Time.Format("2006-01-02")
		}
		classID := ""
		if r.ClassID.Valid {
			classID = r.ClassID.String()
		}
		className := ""
		if r.ClassName.Valid {
			className = r.ClassName.String
		}
		genderStr := fmt.Sprintf("%v", r.Gender)
		items = append(items, StudentListItem{
			StudentID:       r.StudentID.String(),
			AdmissionNumber: r.AdmissionNumber,
			FullName:        r.FullName,
			DateOfBirth:     dobStr,
			Gender:          genderStr,
			ClassID:         classID,
			ClassName:       className,
		})
	}

	return &StudentListResponse{
		Items: items,
		Total: int(total),
	}, nil
}

func (s *studentsService) DeleteStudents(ctx context.Context, schoolID uuid.UUID, studentIDs []uuid.UUID) error {
	if len(studentIDs) == 0 {
		return fmt.Errorf("bad_request: student_ids must be provided and non-empty")
	}

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	uuids := make([]pgtype.UUID, 0, len(studentIDs))
	for _, id := range studentIDs {
		uuids = append(uuids, pgtype.UUID{Bytes: id, Valid: true})
	}

	if err := s.queries.DeleteStudents(ctx, sqlc.DeleteStudentsParams{
		SchoolID:  schoolUUID,
		StudentID: uuids,
	}); err != nil {
		return fmt.Errorf("delete students: %w", err)
	}
	return nil
}
