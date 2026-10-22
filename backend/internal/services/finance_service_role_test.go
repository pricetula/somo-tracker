package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"somotracker/backend/internal/database/sqlc"
)

type financeMockQueries struct {
	*sqlc.Queries
	role    sqlc.UserRole
	err     error
	deleted bool
}

func (m *financeMockQueries) GetSchoolMembershipRole(ctx context.Context, arg sqlc.GetSchoolMembershipRoleParams) (sqlc.UserRole, error) {
	return m.role, m.err
}

func (m *financeMockQueries) DeleteFinance(ctx context.Context, arg sqlc.DeleteFinanceParams) error {
	m.deleted = true
	return nil
}

func TestFinanceService_UserHasAdminRole_Admin(t *testing.T) {
	mq := &financeMockQueries{Queries: nil, role: sqlc.UserRoleADMIN}
	svc := &financeService{queries: mq, logger: zap.NewNop()}
	schoolID := uuid.New()
	userID := uuid.New()
	isAdmin, err := svc.UserHasAdminRole(context.Background(), schoolID, userID)
	require.NoError(t, err)
	require.True(t, isAdmin)
}

func TestFinanceService_UserHasAdminRole_NonAdmin(t *testing.T) {
	mq := &financeMockQueries{Queries: nil, role: sqlc.UserRoleTEACHER}
	svc := &financeService{queries: mq, logger: zap.NewNop()}
	schoolID := uuid.New()
	userID := uuid.New()
	isAdmin, err := svc.UserHasAdminRole(context.Background(), schoolID, userID)
	require.NoError(t, err)
	require.False(t, isAdmin)
}

func TestFinanceService_UserHasAdminRole_QueryError(t *testing.T) {
	mq := &financeMockQueries{Queries: nil, err: context.DeadlineExceeded}
	svc := &financeService{queries: mq, logger: zap.NewNop()}
	schoolID := uuid.New()
	userID := uuid.New()
	isAdmin, err := svc.UserHasAdminRole(context.Background(), schoolID, userID)
	require.Error(t, err)
	require.False(t, isAdmin)
}

func TestFinanceService_DeleteFinance_RejectsNonAdmin(t *testing.T) {
	mq := &financeMockQueries{Queries: nil, role: sqlc.UserRoleTEACHER}
	svc := &financeService{queries: mq, logger: zap.NewNop()}
	schoolID := uuid.New()
	currentUserID := uuid.New()
	userIDs := []uuid.UUID{uuid.New()}
	err := svc.DeleteFinance(context.Background(), schoolID, userIDs, currentUserID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "forbidden")
	require.False(t, mq.deleted)
}

func TestFinanceService_DeleteFinance_AllowsAdmin(t *testing.T) {
	mq := &financeMockQueries{Queries: nil, role: sqlc.UserRoleADMIN}
	svc := &financeService{queries: mq, logger: zap.NewNop()}
	schoolID := uuid.New()
	userID := uuid.New()
	currentUserID := uuid.New()
	userIDs := []uuid.UUID{userID}
	err := svc.DeleteFinance(context.Background(), schoolID, userIDs, currentUserID)
	require.NoError(t, err)
	require.True(t, mq.deleted)
}

func TestFinanceService_DeleteFinance_CannotDeleteSelf(t *testing.T) {
	mq := &financeMockQueries{Queries: nil, role: sqlc.UserRoleADMIN}
	svc := &financeService{queries: mq, logger: zap.NewNop()}
	schoolID := uuid.New()
	currentUserID := uuid.New()
	userIDs := []uuid.UUID{currentUserID}
	err := svc.DeleteFinance(context.Background(), schoolID, userIDs, currentUserID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot delete yourself")
}
