package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// MeResult holds the fields returned by GET /api/me.
type MeResult struct {
	UserName         string `json:"user_name"`
	Email            string `json:"email"`
	ActiveSchoolID   string `json:"active_school_id"`
	SchoolName       string `json:"school_name"`
	ActiveSchoolRole string `json:"active_school_role"`
	TenantID         string `json:"tenant_id"`
}

// MeService describes the current-user retrieval layer.
type MeService interface {
	GetCurrentUser(ctx context.Context, token string, tenantID string) (MeResult, error)
}

type meService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewMeService(pool *pgxpool.Pool, logger *zap.Logger) MeService {
	return &meService{
		pool:   pool,
		logger: logger.With(zap.String("service", "me")),
	}
}

const getMeQuery = `
SELECT 
    u.full_name AS user_name,
    u.email,
    sm.school_id AS active_school_id,
    sch.school_name,
    sm.role AS active_school_role,
    u.tenant_id
FROM 
    sessions s
JOIN 
    users u ON s.user_id = u.id
JOIN 
    school_memberships sm ON u.id = sm.user_id
JOIN 
    schools sch ON sm.school_id = sch.id
WHERE 
    s.token = $1
    AND s.expires_at > NOW()
    AND u.is_active = TRUE
    AND sm.is_active = TRUE;
`

func (s *meService) GetCurrentUser(ctx context.Context, token string, tenantID string) (MeResult, error) {
	if token == "" {
		return MeResult{}, fmt.Errorf("bad_request: missing session token")
	}
	if tenantID == "" {
		return MeResult{}, fmt.Errorf("bad_request: missing tenant context")
	}

	var result MeResult
	txErr := database.WithTenantTx(ctx, s.pool, s.logger, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, getMeQuery, token)
		var schoolID, tenantIDPg pgtype.UUID
		err := row.Scan(
			&result.UserName,
			&result.Email,
			&schoolID,
			&result.SchoolName,
			&result.ActiveSchoolRole,
			&tenantIDPg,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("not_found: session expired or invalid")
		}
		if err != nil {
			return fmt.Errorf("internal_error: failed to retrieve user: %w", err)
		}
		if schoolID.Valid {
			result.ActiveSchoolID = schoolID.String()
		}
		if tenantIDPg.Valid {
			result.TenantID = tenantIDPg.String()
		}
		return nil
	})

	if txErr != nil {
		return MeResult{}, txErr
	}
	return result, nil
}
