package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type TeacherListItem struct {
	MembershipID uuid.UUID `json:"membership_id"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	InvitedAt    *string   `json:"invited_at,omitempty"`
	AcceptedAt   *string   `json:"accepted_at,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    string    `json:"created_at"`
}

type TeacherListResponse struct {
	Items []TeacherListItem `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

type TeacherSummary struct {
	TotalTeachers             int64 `json:"total_teachers"`
	TeachersWithoutAssignment int64 `json:"teachers_without_assignment"`
}

type TeachersService interface {
	ListTeachers(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*TeacherListResponse, error)
	DeleteTeachers(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error
	GetTeacherSummary(ctx context.Context, schoolID uuid.UUID) (*TeacherSummary, error)
}

type teachersService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewTeachersService(queries *sqlc.Queries, logger *zap.Logger) TeachersService {
	return &teachersService{queries: queries, logger: logger.With(zap.String("service", "teachers"))}
}

func (s *teachersService) DeleteTeachers(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error {
	if len(userIDs) == 0 {
		return fmt.Errorf("bad_request: user_ids must be provided and non-empty")
	}
	for _, id := range userIDs {
		if id == currentUserID {
			return fmt.Errorf("bad_request: cannot delete yourself")
		}
	}
	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	userIDsPg := make([]pgtype.UUID, len(userIDs))
	for i, id := range userIDs {
		userIDsPg[i] = pgtype.UUID{Bytes: id, Valid: true}
	}
	if err := s.queries.DeleteTeachers(ctx, sqlc.DeleteTeachersParams{
		SchoolID: schoolUUID,
		Column2:  userIDsPg,
	}); err != nil {
		return fmt.Errorf("delete_teachers exec: %w", err)
	}
	return nil
}

func (s *teachersService) ListTeachers(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*TeacherListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	status := strings.ToLower(strings.TrimSpace(invitationStatus))
	if status != "" && status != "invited" && status != "accepted" && status != "all" {
		return nil, fmt.Errorf("bad_request: invitation_status must be invited|accepted|all")
	}

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	searchParam := strings.TrimSpace(search)

	total, err := s.queries.CountTeachers(ctx, sqlc.CountTeachersParams{
		SchoolID: schoolUUID,
		Column2:  searchParam,
		Column3:  status,
	})
	if err != nil {
		return nil, fmt.Errorf("list_teachers count: %w", err)
	}

	rows, err := s.queries.ListTeachers(ctx, sqlc.ListTeachersParams{
		SchoolID: schoolUUID,
		Column2:  searchParam,
		Column3:  status,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list_teachers query: %w", err)
	}

	items := make([]TeacherListItem, 0, len(rows))
	for _, r := range rows {
		it := TeacherListItem{
			MembershipID: uuid.UUID(r.ID.Bytes),
			UserID:       uuid.UUID(r.UserID.Bytes),
			Email:        r.Email,
			FullName:     r.FullName,
			IsActive:     r.IsActive,
		}
		if r.InvitedAt.Valid {
			s := r.InvitedAt.Time.UTC().Format(time.RFC3339)
			it.InvitedAt = &s
		}
		if r.AcceptedAt.Valid {
			s := r.AcceptedAt.Time.UTC().Format(time.RFC3339)
			it.AcceptedAt = &s
		}
		if r.CreatedAt.Valid {
			it.CreatedAt = r.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		items = append(items, it)
	}

	return &TeacherListResponse{
		Items: items,
		Total: int(total),
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *teachersService) GetTeacherSummary(ctx context.Context, schoolID uuid.UUID) (*TeacherSummary, error) {
	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}

	termRow, err := s.queries.GetCurrentAcademicTermBySchool(ctx, schoolUUID)
	if err != nil {
		return nil, fmt.Errorf("no academic term found for school: %w", err)
	}
	termUUID := pgtype.UUID{Bytes: termRow.ID.Bytes, Valid: true}

	row, err := s.queries.GetTeacherSummary(ctx, sqlc.GetTeacherSummaryParams{
		SchoolID:       schoolUUID,
		AcademicTermID: termUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("get teacher summary: %w", err)
	}

	return &TeacherSummary{
		TotalTeachers:             row.TotalTeachers,
		TeachersWithoutAssignment: row.TeachersWithoutAssignment,
	}, nil
}
