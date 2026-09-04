// Package services contains the business / orchestration layer.
//
// Services call sqlc queries directly. They never touch Fiber, never build
// HTTP responses, and never know about request context other than the standard
// context.Context. This keeps them trivially testable and lets the delivery
// layer (handlers) swap implementations behind interfaces.
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/database/sqlc"
)

// Sentinel errors. Handlers map these to HTTP status codes; services and
// handlers must never construct HTTP responses.
var (
	ErrTenantRequired = errors.New("tenant required")
	ErrInvalidUUID    = errors.New("invalid uuid")
	ErrNotFound       = errors.New("not found")
)

// UserService describes the orchestration layer over the user table.
type UserService interface {
	GetByID(ctx context.Context, tenantID string, id string) (sqlc.User, error)
	GetByEmail(ctx context.Context, tenantID string, email string) (sqlc.User, error)
}

// userService is the concrete implementation. It depends on the sqlc Querier
// (not the pool directly) so it can be unit-tested with a fake Querier.
type userService struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
	logger  *zap.Logger
}

// NewUserService returns a UserService. The pool is required for RLS-backed
// reads (those run inside database.WithTenantTx); non-RLS reads could be
// migrated to use only sqlc.Querier in the future.
func NewUserService(pool *pgxpool.Pool, logger *zap.Logger) UserService {
	return &userService{
		queries: sqlc.New(pool),
		pool:    pool,
		logger:  logger,
	}
}

func (s *userService) GetByID(ctx context.Context, tenantID string, id string) (sqlc.User, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return sqlc.User{}, fmt.Errorf("%w: id", ErrInvalidUUID)
	}

	var user sqlc.User
	txErr := database.WithTenantTx(ctx, s.pool, s.logger, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		var queryErr error
		user, queryErr = q.GetUserByID(ctx, pgtype.UUID{Bytes: [16]byte(parsed)})
		return queryErr
	})
	if txErr != nil {
		return sqlc.User{}, mapSQLCError(txErr, ErrNotFound)
	}
	return user, nil
}

func (s *userService) GetByEmail(ctx context.Context, tenantID string, email string) (sqlc.User, error) {
	if tenantID == "" {
		return sqlc.User{}, ErrTenantRequired
	}
	parsed, err := uuid.Parse(tenantID)
	if err != nil {
		return sqlc.User{}, fmt.Errorf("%w: tenant_id", ErrInvalidUUID)
	}

	var user sqlc.User
	txErr := database.WithTenantTx(ctx, s.pool, s.logger, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		var queryErr error
		user, queryErr = q.GetUserByEmail(ctx, sqlc.GetUserByEmailParams{
			Email:    email,
			TenantID: pgtype.UUID{Bytes: [16]byte(parsed)},
		})
		return queryErr
	})
	if txErr != nil {
		return sqlc.User{}, mapSQLCError(txErr, ErrNotFound)
	}
	return user, nil
}
