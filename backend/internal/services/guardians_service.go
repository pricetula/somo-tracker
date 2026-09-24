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
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewGuardiansService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) GuardiansService {
	return &guardiansService{pool: pool, queries: queries, logger: logger.With(zap.String("service", "guardians"))}
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
	query := `DELETE FROM school_memberships WHERE school_id = $1 AND user_id = ANY($2) AND role = 'GUARDIAN'`
	_, err := s.pool.Exec(ctx, query, schoolID, userIDs)
	if err != nil {
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

	where := []string{
		"sm.school_id = $1",
		"sm.role = 'GUARDIAN'",
	}
	args := []any{schoolID}
	argIdx := 2

	if strings.TrimSpace(search) != "" {
		where = append(where, fmt.Sprintf("(u.email ILIKE $%d OR u.full_name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	switch strings.ToLower(invitationStatus) {
	case "invited":
		where = append(where, "sm.invited_at IS NOT NULL AND sm.accepted_at IS NULL")
	case "accepted":
		where = append(where, "sm.accepted_at IS NOT NULL")
	case "all", "":
		// no extra filter
	default:
		return nil, fmt.Errorf("bad_request: invitation_status must be invited|accepted|all")
	}

	whereSQL := strings.Join(where, " AND ")

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM school_memberships sm JOIN users u ON u.id = sm.user_id WHERE %s`, whereSQL)
	var total int
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("list_guardians count: %w", err)
	}

	itemsSQL := fmt.Sprintf(`
		SELECT sm.id, sm.user_id, u.email, u.full_name, sm.invited_at, sm.accepted_at, sm.is_active, sm.created_at
		FROM school_memberships sm
		JOIN users u ON u.id = sm.user_id
		WHERE %s
		ORDER BY sm.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, itemsSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list_guardians query: %w", err)
	}
	defer rows.Close()

	items := make([]GuardianListItem, 0, limit)
	for rows.Next() {
		var it GuardianListItem
		var invitedAt, acceptedAt, createdAt *time.Time
		var isActive bool
		if err := rows.Scan(&it.MembershipID, &it.UserID, &it.Email, &it.FullName, &invitedAt, &acceptedAt, &isActive, &createdAt); err != nil {
			return nil, fmt.Errorf("list_guardians scan: %w", err)
		}
		if invitedAt != nil {
			s := invitedAt.UTC().Format(time.RFC3339)
			it.InvitedAt = &s
		}
		if acceptedAt != nil {
			s := acceptedAt.UTC().Format(time.RFC3339)
			it.AcceptedAt = &s
		}
		it.IsActive = isActive
		if createdAt != nil {
			s := createdAt.UTC().Format(time.RFC3339)
			it.CreatedAt = s
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list_guardians rows err: %w", err)
	}

	return &GuardianListResponse{
		Items: items,
		Total: total,
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
