package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type GuardianInvitationServiceInterface interface {
	CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error)
	CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []InvitationItem) (uuid.UUID, error)
	InsertItems(ctx context.Context, jobID uuid.UUID, items []InvitationItem) error
	GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error)
	GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error)
	GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result []byte, lastError string, attempts int) error
	UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result []byte, status string, lastError string) error
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error
	IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error
	GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error)
	GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error)
	UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error)
	UserHasGuardianRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error)
	ProvisionInvitee(ctx context.Context, tenantID, schoolID, invitedBy uuid.UUID, email, fullName, role, stytchMemberID string) error
}

type GuardianInvitationService struct {
	*GenericInvitationService
}

func NewGuardianInvitationService(pool *pgxpool.Pool, logger *zap.Logger) GuardianInvitationServiceInterface {
	return &GuardianInvitationService{
		GenericInvitationService: NewGenericInvitationService(pool, logger, "GUARDIAN_INVITATION", "GUARDIAN"),
	}
}

func (s *GuardianInvitationService) UserHasGuardianRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error) {
	hasAdmin, err := s.UserHasAdminRole(ctx, schoolID, userID)
	if err != nil {
		return false, err
	}
	return hasAdmin, nil
}
