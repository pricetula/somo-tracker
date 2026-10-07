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

type FinanceListItem struct {
	MembershipID uuid.UUID `json:"membership_id"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	InvitedAt    *string   `json:"invited_at,omitempty"`
	AcceptedAt   *string   `json:"accepted_at,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    string    `json:"created_at"`
}

type FinanceListResponse struct {
	Items []FinanceListItem `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

type FinanceService interface {
	ListFinance(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*FinanceListResponse, error)
	DeleteFinance(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error
}

type financeService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewFinanceService(queries *sqlc.Queries, logger *zap.Logger) FinanceService {
	return &financeService{queries: queries, logger: logger.With(zap.String("service", "finance"))}
}

func (s *financeService) DeleteFinance(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error {
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
	if err := s.queries.DeleteFinance(ctx, sqlc.DeleteFinanceParams{
		SchoolID: schoolUUID,
		Column2:  userIDsPg,
	}); err != nil {
		return fmt.Errorf("delete_finance exec: %w", err)
	}
	return nil
}

func (s *financeService) ListFinance(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*FinanceListResponse, error) {
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

	total, err := s.queries.CountFinance(ctx, sqlc.CountFinanceParams{
		SchoolID: schoolUUID,
		Column2:  searchParam,
		Column3:  status,
	})
	if err != nil {
		return nil, fmt.Errorf("list_finance count: %w", err)
	}

	rows, err := s.queries.ListFinance(ctx, sqlc.ListFinanceParams{
		SchoolID: schoolUUID,
		Column2:  searchParam,
		Column3:  status,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list_finance query: %w", err)
	}

	items := make([]FinanceListItem, 0, len(rows))
	for _, r := range rows {
		it := FinanceListItem{
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

	return &FinanceListResponse{
		Items: items,
		Total: int(total),
		Page:  page,
		Limit: limit,
	}, nil
}
