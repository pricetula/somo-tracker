package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"somotracker/backend/internal/database/sqlc"
)

type InvitationItem struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
}

type BulkJob struct {
	ID             uuid.UUID       `json:"id"`
	JobType        string          `json:"job_type"`
	IdempotencyKey string          `json:"idempotency_key"`
	SchoolID       uuid.UUID       `json:"school_id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	CreatedBy      uuid.UUID       `json:"created_by"`
	Status         string          `json:"status"`
	TotalRecords   int             `json:"total_records"`
	SucceededCount int             `json:"succeeded_count"`
	FailedCount    int             `json:"failed_count"`
	DeferredCount  int             `json:"deferred_count"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type BulkJobItem struct {
	ID           uuid.UUID       `json:"id"`
	JobID        uuid.UUID       `json:"job_id"`
	RowIndex     int             `json:"row_index"`
	Payload      json.RawMessage `json:"payload"`
	Result       json.RawMessage `json:"result"`
	Status       string          `json:"status"`
	AttemptCount int             `json:"attempt_count"`
	LastError    string          `json:"last_error"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type AdminInvitationService interface {
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
	ProvisionInvitee(ctx context.Context, tenantID, schoolID, invitedBy uuid.UUID, email, fullName, role, stytchMemberID string) error
}

type adminInvitationService struct {
	*GenericInvitationService
}

func NewAdminInvitationService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) AdminInvitationService {
	return &adminInvitationService{GenericInvitationService: NewGenericInvitationService(pool, logger.With(zap.String("service", "admin_invitation")), "ADMIN_INVITATION", "ADMIN")}
}
