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

type GuardianListItem struct {
	MembershipID uuid.UUID `json:"membership_id"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	InvitedAt    *string   `json:"invited_at,omitempty"`
	AcceptedAt   *string   `json:"accepted_at,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    string    `json:"created_at"`
}

type GuardianListResponse struct {
	Items []GuardianListItem `json:"items"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

type GuardianSummary struct {
	TotalGuardians          int64 `json:"total_guardians"`
	GuardiansWithoutStudent int64 `json:"guardians_without_student"`
}

type GuardiansService interface {
	ListGuardians(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*GuardianListResponse, error)
	DeleteGuardians(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error
	GetGuardianSummary(ctx context.Context, schoolID uuid.UUID) (*GuardianSummary, error)
}

type guardiansService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewGuardiansService(queries *sqlc.Queries, logger *zap.Logger) GuardiansService {
	return &guardiansService{queries: queries, logger: logger.With(zap.String("service", "guardians"))}
}

func (s *guardiansService) DeleteGuardians(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error {
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
	if err := s.queries.DeleteGuardians(ctx, sqlc.DeleteGuardiansParams{
		SchoolID: schoolUUID,
		Column2:  userIDsPg,
	}); err != nil {
		return fmt.Errorf("delete_guardians exec: %w", err)
	}
	return nil
}

func (s *guardiansService) ListGuardians(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*GuardianListResponse, error) {
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

	total, err := s.queries.CountGuardians(ctx, sqlc.CountGuardiansParams{
		SchoolID: schoolUUID,
		Column2:  searchParam,
		Column3:  status,
	})
	if err != nil {
		return nil, fmt.Errorf("list_guardians count: %w", err)
	}

	rows, err := s.queries.ListGuardians(ctx, sqlc.ListGuardiansParams{
		SchoolID: schoolUUID,
		Column2:  searchParam,
		Column3:  status,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list_guardians query: %w", err)
	}

	items := make([]GuardianListItem, 0, len(rows))
	for _, r := range rows {
		it := GuardianListItem{
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

	return &GuardianListResponse{
		Items: items,
		Total: int(total),
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *guardiansService) GetGuardianSummary(ctx context.Context, schoolID uuid.UUID) (*GuardianSummary, error) {
	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	row, err := s.queries.GetGuardianSummary(ctx, schoolUUID)
	if err != nil {
		return nil, fmt.Errorf("get guardian summary: %w", err)
	}
	return &GuardianSummary{
		TotalGuardians:          row.TotalGuardians,
		GuardiansWithoutStudent: row.GuardiansWithoutStudent,
	}, nil
}
