package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type AdminListItem struct {
	MembershipID uuid.UUID `json:"membership_id"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	InvitedAt    *string   `json:"invited_at,omitempty"`
	AcceptedAt   *string   `json:"accepted_at,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    string    `json:"created_at"`
}

type AdminListResponse struct {
	Items []AdminListItem `json:"items"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

type AdminsService interface {
	ListAdmins(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*AdminListResponse, error)
	DeleteAdmins(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error
}

type adminsService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewAdminsService(pool *pgxpool.Pool, logger *zap.Logger) AdminsService {
	return &adminsService{pool: pool, logger: logger.With(zap.String("service", "admins"))}
}

func (s *adminsService) DeleteAdmins(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) error {
	if len(userIDs) == 0 {
		return fmt.Errorf("bad_request: user_ids must be provided and non-empty")
	}
	for _, id := range userIDs {
		if id == currentUserID {
			return fmt.Errorf("bad_request: cannot delete yourself")
		}
	}
	query := `DELETE FROM school_memberships WHERE school_id = $1 AND user_id = ANY($2)`
	_, err := s.pool.Exec(ctx, query, schoolID, userIDs)
	if err != nil {
		return fmt.Errorf("delete_admins exec: %w", err)
	}
	return nil
}

func (s *adminsService) ListAdmins(ctx context.Context, schoolID uuid.UUID, page, limit int, search string, invitationStatus string) (*AdminListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	where := []string{
		"sm.school_id = $1",
		"sm.role = 'ADMIN'",
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
		return nil, fmt.Errorf("list_admins count: %w", err)
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
		return nil, fmt.Errorf("list_admins query: %w", err)
	}
	defer rows.Close()

	items := make([]AdminListItem, 0, limit)
	for rows.Next() {
		var it AdminListItem
		var invitedAt, acceptedAt, createdAt *time.Time
		var isActive bool
		if err := rows.Scan(&it.MembershipID, &it.UserID, &it.Email, &it.FullName, &invitedAt, &acceptedAt, &isActive, &createdAt); err != nil {
			return nil, fmt.Errorf("list_admins scan: %w", err)
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
		return nil, fmt.Errorf("list_admins rows err: %w", err)
	}

	return &AdminListResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
