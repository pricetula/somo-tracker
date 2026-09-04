package services

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

// TenantService describes the orchestration layer over the tenant table.
// Tenant lookups do not go through WithTenantTx (no RLS), so this service is
// pure sqlc.Querier with no pool dependency.
type TenantService interface {
	GetBySlug(ctx context.Context, slug string) (sqlc.Tenant, error)
}

type tenantService struct {
	queries sqlc.Querier
	logger  *zap.Logger
}

func NewTenantService(queries sqlc.Querier, logger *zap.Logger) TenantService {
	return &tenantService{queries: queries, logger: logger}
}

func (s *tenantService) GetBySlug(ctx context.Context, slug string) (sqlc.Tenant, error) {
	t, err := s.queries.GetTenantBySlug(ctx, slug)
	if err != nil {
		return sqlc.Tenant{}, mapSQLCError(err, ErrNotFound)
	}
	return t, nil
}

// mapSQLCError converts a sqlc error into a sentinel. The mapping currently
// collapses every "row not found" pgx error to ErrNotFound; future work can
// narrow this with pgx.ErrNoRows checks.
func mapSQLCError(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, notFound) {
		return err
	}
	// sqlc queries return pgx.ErrNoRows for missing records. Without
	// importing pgx here we fall back to substring match as a transitional
	// approach; a follow-up commit can replace this with errors.Is.
	if err.Error() == "no rows in result set" {
		return notFound
	}
	return err
}
