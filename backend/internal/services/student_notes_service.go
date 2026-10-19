package services

import (
	"context"

	"github.com/google/uuid"
	"somotracker/backend/internal/database/sqlc"
)

type StudentNote struct {
	ID        uuid.UUID `json:"id"`
	SchoolID  uuid.UUID `json:"school_id"`
	StudentID uuid.UUID `json:"student_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type StudentNotesService struct {
	queries *sqlc.Queries
}

func NewStudentNotesService(queries *sqlc.Queries) *StudentNotesService {
	return &StudentNotesService{queries: queries}
}

func (s *StudentNotesService) List(ctx context.Context, schoolID, studentID uuid.UUID, page, limit int) ([]StudentNote, error) {
	offset := (page - 1) * limit
	rows, err := s.queries.ListStudentNotes(ctx, sqlc.ListStudentNotesParams{
		SchoolID:  uuidToPG(schoolID),
		StudentID: uuidToPG(studentID),
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	var notes []StudentNote
	for _, r := range rows {
		notes = append(notes, StudentNote{
			ID: pgToUUID(r.ID), SchoolID: pgToUUID(r.SchoolID), StudentID: pgToUUID(r.StudentID),
			AuthorID: pgToUUID(r.AuthorID), Content: r.Content,
			CreatedAt: r.CreatedAt.Time.String(), UpdatedAt: r.UpdatedAt.Time.String(),
		})
	}
	return notes, nil
}
