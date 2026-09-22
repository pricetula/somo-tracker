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
}

type StudentListResponse struct {
	Items []StudentListItem `json:"items"`
	Total int               `json:"total"`
}

type StudentsService interface {
	ListStudents(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string) (*StudentListResponse, error)
}

type studentsService struct {
	queries *sqlc.Queries
}

func NewStudentsService(queries *sqlc.Queries) StudentsService {
	return &studentsService{queries: queries}
}

func (s *studentsService) ListStudents(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string) (*StudentListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	rows, err := s.queries.ListStudents(ctx, sqlc.ListStudentsParams{
		SchoolID: schoolUUID,
		Column2:  search,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}

	total, err := s.queries.CountStudents(ctx, sqlc.CountStudentsParams{
		SchoolID: schoolUUID,
		Column2:  search,
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
		items = append(items, StudentListItem{
			StudentID:       r.StudentID.String(),
			AdmissionNumber: r.AdmissionNumber,
			FullName:        r.FullName,
			DateOfBirth:     dobStr,
			Gender:          r.Gender,
		})
	}

	return &StudentListResponse{
		Items: items,
		Total: int(total),
	}, nil
}
