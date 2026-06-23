package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// ─── Task types ──────────────────────────────────────────────────────────

const (
	TypeProcessImport = "imports:process"
)

// ProcessImportPayload is the payload for the Asynq task.
type ProcessImportPayload struct {
	ImportJobID string              `json:"import_job_id"`
	TenantID    string              `json:"tenant_id"`
	SchoolID    string              `json:"school_id"`
	Role        string              `json:"role"`
	Records     []ImportStaffRecord `json:"records"`

	// Stytch org ID resolved from tenant before enqueue
	StytchOrgID string `json:"stytch_org_id"`
	// Frontend URL for invite redirect
	BackendURL string `json:"backend_url"`
	// ParentImportJobID links correction jobs to the original import
	ParentImportJobID string `json:"parent_import_job_id,omitempty"`
}

// ─── Worker ──────────────────────────────────────────────────────────────

// Worker handles Asynq task processing for bulk imports.
type Worker struct {
	repo   Repository
	rdb    ProgressPublisher
	idp    auth.IdentityProvider
	logger *zap.Logger
	cfg    config.Config
}

// NewWorker creates a new Worker.
func NewWorker(pools *database.Pools, repo Repository, idp auth.IdentityProvider, cfg config.Config, logger *zap.Logger) *Worker {
	return &Worker{
		repo:   repo,
		rdb:    pools.Redis,
		idp:    idp,
		logger: logger,
		cfg:    cfg,
	}
}

// ProcessImport is the Asynq handler for bulk staff import processing.
func (w *Worker) ProcessImport(ctx context.Context, t *asynq.Task) error {
	var payload ProcessImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger := w.logger.With(
		zap.String("import_job_id", payload.ImportJobID),
		zap.String("role", payload.Role),
		zap.Int("total_records", len(payload.Records)),
	)

	logger.Info("starting import job processing")

	// Mark job as started
	if err := w.repo.SetImportJobStarted(ctx, payload.ImportJobID); err != nil {
		logger.Error("failed to mark job started", zap.Error(err))
		return fmt.Errorf("set started: %w", err)
	}

	// Publish initial progress (non-fatal if Redis is down)
	w.publishProgress(ctx, payload.ImportJobID, "processing", 0, 0, 0, len(payload.Records))

	overallSuccess := 0
	overallFailed := 0
	overallProcessed := 0
	hasErrors := false

	// STAGE 1: Bulk DB ingestion (idempotent — ON CONFLICT DO NOTHING)
	for i := 0; i < len(payload.Records); i += BatchSize {
		// Check for task cancellation (e.g. Asynq timeout)
		select {
		case <-ctx.Done():
			logger.Warn("stage 1 cancelled", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		end := i + BatchSize
		if end > len(payload.Records) {
			end = len(payload.Records)
		}
		batch := payload.Records[i:end]

		logger.Info("processing stage 1 batch",
			zap.Int("batch_start", i),
			zap.Int("batch_end", end),
		)

		now := time.Now().UTC()

		var inserted map[string]string // temp_id -> invitation_id
		var failures []FailedInsertion
		var stage1Err error

		if payload.ParentImportJobID != "" {
			// Correction path: update existing invitation rows in-place
			var updated int
			updated, stage1Err = w.repo.BulkUpdateInvitations(
				ctx, batch, payload.Role, payload.ImportJobID, now,
			)
			if stage1Err == nil {
				inserted = make(map[string]string, len(batch))
				for _, rec := range batch {
					inserted[rec.TempID] = rec.TempID
				}
				logger.Info("correction batch updated",
					zap.Int("batch_size", len(batch)),
					zap.Int("updated", updated),
				)
			}
		} else {
			// Normal path: fresh insert of invitations
			inserted, failures, stage1Err = w.repo.BulkInsertInvitations(
				ctx, batch, payload.TenantID, payload.SchoolID,
				payload.Role, payload.ImportJobID, now, payload.ImportJobID+"_",
			)
		}

		if stage1Err != nil {
			logger.Error("stage 1 batch failed", zap.Error(stage1Err))
			for _, rec := range batch {
				raw, _ := json.Marshal(rec)
				if err := w.repo.RecordImportFailure(ctx, payload.ImportJobID, string(raw), stage1Err.Error()); err != nil {
					logger.Error("failed to record import failure", zap.Error(err))
				}
			}
			overallFailed += len(batch)
			hasErrors = true
			continue
		}

		// Count duplicates as failed (normal path only)
		for _, f := range failures {
			logger.Info("duplicate skipped",
				zap.String("email", f.Email),
				zap.String("reason", f.Reason),
			)
			overallFailed++
			hasErrors = true
		}

		overallProcessed += len(batch)
	}

	logger.Info("stage 1 complete",
		zap.Int("records_attempted", len(payload.Records)),
	)

	// STAGE 2: Query DB for unprocessed records (safe on retry — skips
	// records that already have a stytch_member_id from a previous run).
	stage2Input, err := w.repo.GetPendingStage2Records(ctx, payload.ImportJobID)
	if err != nil {
		logger.Error("failed to query pending stage 2 records", zap.Error(err))
		return fmt.Errorf("get pending stage 2 records: %w", err)
	}

	logger.Info("stage 2 candidates",
		zap.Int("pending_invitations", len(stage2Input)),
	)

	if len(stage2Input) > 0 {
		w.processStage2(ctx, &payload, stage2Input, &overallSuccess, &overallFailed, &hasErrors, logger)
	}

	// Finalize job
	finalStatus := "completed"
	if hasErrors {
		finalStatus = "completed_with_errors"
	}

	if err := w.repo.UpdateImportJobStatus(ctx, payload.ImportJobID, finalStatus,
		overallProcessed, overallSuccess, overallFailed); err != nil {
		logger.Error("failed to finalize job status", zap.Error(err))
	}
	if err := w.repo.SetImportJobCompleted(ctx, payload.ImportJobID, hasErrors); err != nil {
		logger.Error("failed to set job completed", zap.Error(err))
	}

	logger.Info("import job completed",
		zap.String("status", finalStatus),
		zap.Int("success", overallSuccess),
		zap.Int("failed", overallFailed),
	)

	// Broadcast finished event (non-fatal if Redis is down)
	w.publishFinished(ctx, payload.ImportJobID, finalStatus, overallProcessed, overallSuccess, overallFailed, len(payload.Records))

	return nil
}

// HandleError implements asynq.ErrorHandler. It is called when a task has
// exhausted all retries (MaxRetry(3)). It updates the import job's status to
// 'failed' so the job is not left stuck as 'processing'.
func (w *Worker) HandleError(ctx context.Context, task *asynq.Task, err error) {
	var payload ProcessImportPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error("asynq error handler: failed to unmarshal payload",
			zap.Error(err),
		)
		return
	}

	w.logger.Warn("asynq task failed after all retries",
		zap.String("import_job_id", payload.ImportJobID),
		zap.Error(err),
	)

	if repoErr := w.repo.SetImportJobFailed(ctx, payload.ImportJobID); repoErr != nil {
		w.logger.Error("asynq error handler: failed to set job status to failed",
			zap.String("import_job_id", payload.ImportJobID),
			zap.Error(repoErr),
		)
	}
}

// Stage2Record holds data needed for the Stytch invitation step.
type Stage2Record struct {
	InvitationID string
	Email        string
	FirstName    string
	LastName     string
}

// processStage2 sends Stytch invite emails with bounded concurrency.
// Re-entry safe: each record checks stytch_member_id before sending, and
// skips already-invited records. On task retry, only records without a
// stytch_member_id are queried by the caller.
func (w *Worker) processStage2(
	ctx context.Context,
	payload *ProcessImportPayload,
	records []Stage2Record,
	successCount *int,
	failedCount *int,
	hasErrors *bool,
	logger *zap.Logger,
) {
	var mu sync.Mutex
	sem := make(chan struct{}, StytchConcurrency)
	var wg sync.WaitGroup

	for _, rec := range records {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(rec Stage2Record) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			// Idempotency guard: skip if already has stytch_member_id.
			// This handles the edge case where a record was invited in a
			// previous run but stytch_member_id was written just before a crash.
			existingMemberID, err := w.repo.GetInvitationStytchMemberID(ctx, rec.InvitationID)
			if err != nil {
				logger.Error("idempotency check failed", zap.String("invitation_id", rec.InvitationID), zap.Error(err))
				mu.Lock()
				*failedCount++
				*hasErrors = true
				mu.Unlock()
				return
			}
			if existingMemberID != "" {
				logger.Info("skipping already-invited record",
					zap.String("email", rec.Email),
					zap.String("stytch_member_id", existingMemberID),
				)
				mu.Lock()
				*successCount++
				mu.Unlock()
				return
			}

			// Build full name
			fullName := rec.FirstName
			if rec.LastName != "" {
				if fullName != "" {
					fullName += " "
				}
				fullName += rec.LastName
			}

			// Send Stytch invite with retry
			var memberID string
			var lastErr error
		retryLoop:
			for attempt := 1; attempt <= StytchMaxRetries; attempt++ {
				memberID, lastErr = w.idp.InviteMemberByEmail(
					ctx, payload.StytchOrgID, rec.Email, fullName, payload.BackendURL+"/api/auth/invite/callback",
				)
				if lastErr == nil {
					break
				}
				// Check if permanent error (4xx other than rate-limit)
				if isPermanentStytchError(lastErr) {
					logger.Warn("permanent stytch error",
						zap.String("email", rec.Email),
						zap.Error(lastErr),
					)
					break
				}
				// Transient error — backoff before retry
				backoff := time.Duration(attempt*2) * time.Second
				logger.Warn("transient stytch error, retrying",
					zap.String("email", rec.Email),
					zap.Int("attempt", attempt),
					zap.Duration("backoff", backoff),
					zap.Error(lastErr),
				)
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					lastErr = ctx.Err()
					break retryLoop
				}
			}

			mu.Lock()
			defer mu.Unlock()

			if lastErr == nil && memberID != "" {
				// Success — store Stytch member ID
				if err := w.repo.SetInvitationStytchMemberID(ctx, rec.InvitationID, memberID); err != nil {
					logger.Error("failed to persist stytch member id",
						zap.String("invitation_id", rec.InvitationID),
						zap.Error(err),
					)
				}
				*successCount++
			} else {
				// Permanent failure
				errMsg := "stytch invite failed"
				if lastErr != nil {
					errMsg = lastErr.Error()
				}
				if err := w.repo.SetInvitationFailed(ctx, rec.InvitationID, errMsg, StytchMaxRetries); err != nil {
					logger.Error("failed to persist invitation failure",
						zap.String("invitation_id", rec.InvitationID),
						zap.Error(err),
					)
				}
				*failedCount++
				*hasErrors = true
			}

			// Publish progress update (non-fatal if Redis is down)
			w.publishProgress(ctx, payload.ImportJobID, "processing",
				*successCount+*failedCount, *successCount, *failedCount, len(payload.Records))
		}(rec)
	}

	wg.Wait()
}

// isPermanentStytchError checks if a Stytch error is permanent (non-retryable).
func isPermanentStytchError(err error) bool {
	errStr := err.Error()
	// Common permanent Stytch errors
	permanentIndicators := []string{
		"invalid_email",
		"email_invalid",
		"blocked_domain",
		"domain_not_allowed",
		"member_already_exists",
		"not_found",
	}
	for _, indicator := range permanentIndicators {
		if strings.Contains(strings.ToLower(errStr), indicator) {
			return true
		}
	}
	return false
}

// ─── Progress Publishing ─────────────────────────────────────────────────

// publishProgress publishes a progress event. Failures are logged but never
// returned — Redis unavailability must not block the import pipeline.
func (w *Worker) publishProgress(ctx context.Context, jobID, status string, processed, success, failed, total int) {
	event := ImportProgressEvent{
		Type:             EventProgress,
		ImportJobID:      jobID,
		Status:           status,
		ProcessedRecords: processed,
		SuccessCount:     success,
		FailedCount:      failed,
		TotalRecords:     total,
	}
	w.publishEvent(ctx, jobID, event)
}

// publishFinished publishes a finished event. Same non-fatal semantics as
// publishProgress.
func (w *Worker) publishFinished(ctx context.Context, jobID, status string, processed, success, failed, total int) {
	event := ImportProgressEvent{
		Type:             EventFinished,
		ImportJobID:      jobID,
		Status:           status,
		ProcessedRecords: processed,
		SuccessCount:     success,
		FailedCount:      failed,
		TotalRecords:     total,
	}
	w.publishEvent(ctx, jobID, event)
}

// publishEvent serialises and publishes an event to Redis pub/sub.
// If Redis is unavailable the error is logged and the call returns without
// interrupting the caller — the SSE polling fallback will pick up the
// progress from the database.
func (w *Worker) publishEvent(ctx context.Context, jobID string, event ImportProgressEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		w.logger.Error("failed to marshal progress event", zap.Error(err))
		return
	}
	if err := w.rdb.Publish(ctx, RedisChannelProgress+jobID, string(data)).Err(); err != nil {
		w.logger.Warn("redis publish failed (non-fatal)",
			zap.String("channel", RedisChannelProgress+jobID),
			zap.Error(err),
		)
	}
}

// ─── Create Asynq Client ─────────────────────────────────────────────────

func NewAsynqClient(rdb *redis.Client) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr: rdb.Options().Addr,
	})
}

// ─── Create Asynq Server ─────────────────────────────────────────────────

func NewAsynqServer(rdb *redis.Client, cfg config.Config, errorHandler asynq.ErrorHandler) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr: rdb.Options().Addr,
		},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			StrictPriority: false,
			Logger:         asynqLogger{},
			ErrorHandler:   errorHandler,
		},
	)
}

// asynqLogger adapts log/slog to asynq.Logger interface.
type asynqLogger struct{}

func (asynqLogger) Debug(args ...interface{}) {}
func (asynqLogger) Info(args ...interface{}) {
	slog.Info(fmt.Sprint(args...))
}
func (asynqLogger) Warn(args ...interface{}) {
	slog.Warn(fmt.Sprint(args...))
}
func (asynqLogger) Error(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
}
func (asynqLogger) Fatal(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
}
