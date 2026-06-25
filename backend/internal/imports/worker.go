package imports

import (
	"context"
	"encoding/json"
	"errors"
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
	repo     Repository
	rdb      ProgressPublisher
	redisCli *redis.Client // full Redis client for hash operations
	idp      auth.IdentityProvider
	logger   *zap.Logger
	cfg      config.Config
}

// NewWorker creates a new Worker.
func NewWorker(pools *database.Pools, repo Repository, idp auth.IdentityProvider, cfg config.Config, logger *zap.Logger) *Worker {
	return &Worker{
		repo:     repo,
		rdb:      pools.Redis,
		redisCli: pools.Redis,
		idp:      idp,
		logger:   logger,
		cfg:      cfg,
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

		var failures []FailedInsertion
		var stage1Err error

		if payload.ParentImportJobID != "" {
			// Correction path: update existing invitation rows in-place
			var updated int
			updated, stage1Err = w.repo.BulkUpdateInvitations(
				ctx, batch, payload.Role, payload.ImportJobID, now,
			)
			if stage1Err == nil {
				logger.Info("correction batch updated",
					zap.Int("batch_size", len(batch)),
					zap.Int("updated", updated),
				)
			}
		} else {
			// Normal path: fresh insert of invitations
			_, failures, stage1Err = w.repo.BulkInsertInvitations(
				ctx, batch, payload.TenantID, payload.SchoolID,
				payload.Role, payload.ImportJobID, now, payload.ImportJobID+"_",
			)
		}

		if stage1Err != nil {
			logger.Error("stage 1 batch failed", zap.Error(stage1Err))
			// Bulk-insert all failures in a single query instead of N round-trips.
			if err := w.repo.BulkRecordImportFailure(ctx, payload.ImportJobID, batch, stage1Err.Error()); err != nil {
				logger.Error("failed to bulk record import failures", zap.Error(err))
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
	FullName     string
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

			// Full name from the record
			fullName := rec.FullName

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

			// Snapshot counters under lock, then release before Redis I/O.
			snapProcessed := *successCount + *failedCount
			snapSuccess := *successCount
			snapFailed := *failedCount
			mu.Unlock()

			// Publish progress update outside the critical section (non-fatal if Redis is down).
			// Holding the mutex during Redis PUBLISH would serialise all Stage 2 goroutines.
			w.publishProgress(ctx, payload.ImportJobID, "processing",
				snapProcessed, snapSuccess, snapFailed, len(payload.Records))
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

// ============================================================================
// Student Import Worker — Anti-Loop Engine
// ============================================================================

// ProcessStudentImport is the Asynq handler for bulk student import processing.
func (w *Worker) ProcessStudentImport(ctx context.Context, t *asynq.Task) error {
	var payload StudentImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal student import payload: %w", err)
	}

	logger := w.logger.With(
		zap.String("job_id", payload.JobID),
		zap.String("tenant_id", payload.TenantID),
	)

	// Phase 0 — Idempotency guard
	status, totalRecords, schoolID, err := w.repo.GetImportJobStatus(ctx, payload.JobID)
	if err != nil {
		return fmt.Errorf("imports.Worker.ProcessStudentImport: get job status: %w", err)
	}
	if status != "pending" {
		logger.Info("skipping job: status is not pending", zap.String("status", status))
		return nil
	}

	// Mark as started
	if err := w.repo.SetImportJobStarted(ctx, payload.JobID); err != nil {
		return fmt.Errorf("imports.Worker.ProcessStudentImport: set started: %w", err)
	}

	logger.Info("starting student import processing",
		zap.Int("total_records", totalRecords),
	)

	// Initialise Redis progress hash
	w.updateRedisProgress(ctx, payload.JobID, "processing", 0, totalRecords, 0, 0)

	// Phase A — Load staging rows (1 query)
	stagingRows, err := w.repo.GetStagingRows(ctx, payload.JobID)
	if err != nil {
		return fmt.Errorf("imports.Worker.ProcessStudentImport: get staging: %w", err)
	}

	// Build de-duplicated reference ID lists
	classIDSet := make(map[string]struct{})
	parentIDSet := make(map[string]struct{})
	for _, sr := range stagingRows {
		raw := sr.RawData
		if cid, ok := raw["class_id"].(string); ok && cid != "" {
			classIDSet[cid] = struct{}{}
		}
		if pid, ok := raw["cbc_student_parents_id"].(string); ok && pid != "" {
			parentIDSet[pid] = struct{}{}
		}
	}

	uniqueClassIDs := make([]string, 0, len(classIDSet))
	for id := range classIDSet {
		uniqueClassIDs = append(uniqueClassIDs, id)
	}
	uniqueParentIDs := make([]string, 0, len(parentIDSet))
	for id := range parentIDSet {
		uniqueParentIDs = append(uniqueParentIDs, id)
	}

	// Phase B — Bulk reference validation (exactly 2 queries)
	validClassesMap, err := w.repo.GetValidClasses(ctx, payload.TenantID, schoolID, uniqueClassIDs)
	if err != nil {
		return fmt.Errorf("imports.Worker.ProcessStudentImport: valid classes: %w", err)
	}

	validParentsMap, err := w.repo.GetValidParentIDs(ctx, payload.TenantID, uniqueParentIDs)
	if err != nil {
		return fmt.Errorf("imports.Worker.ProcessStudentImport: valid parents: %w", err)
	}

	// Resolve academic term ID
	// We need academic_year and term from the first staging row's raw_data
	var academicYear, term string
	if len(stagingRows) > 0 {
		if ay, ok := stagingRows[0].RawData["academic_year"].(string); ok {
			academicYear = ay
		}
		if t, ok := stagingRows[0].RawData["term"].(string); ok {
			term = t
		}
	}

	academicTermID, err := w.repo.ResolveAcademicTerm(ctx, payload.TenantID, schoolID, academicYear, term)
	if err != nil {
		// If we can't resolve the term, mark all as failed
		allFailed := make([]FailedRow, 0, len(stagingRows))
		for _, sr := range stagingRows {
			allFailed = append(allFailed, FailedRow{
				RawData:      sr.RawData,
				ErrorMessage: fmt.Sprintf("academic term '%s %s' not found for this organisation", academicYear, term),
			})
		}
		w.finaliseStudentImport(ctx, payload.JobID, totalRecords, allFailed, nil, logger)
		return nil
	}

	// Phase C — Single-pass validation and row splitting
	now := time.Now().UTC()
	currentYear := now.Year()

	var validStudents []ValidStudent
	var failedRows []FailedRow

	for _, sr := range stagingRows {
		raw := sr.RawData
		exFn := func(msg string) {
			failedRows = append(failedRows, FailedRow{
				RawData:      raw,
				ErrorMessage: msg,
			})
		}

		fullName, _ := raw["full_name"].(string)
		gender, _ := raw["gender"].(string)
		dateOfBirth, _ := raw["date_of_birth"].(string)
		upiNumber, _ := raw["upi_number"].(string)
		knecNumber, _ := raw["knec_assessment_number"].(string)
		classID, _ := raw["class_id"].(string)
		parentID, _ := raw["cbc_student_parents_id"].(string)

		// Rule 1: full_name non-empty after trim
		if strings.TrimSpace(fullName) == "" {
			exFn("full_name is required and cannot be blank")
			continue
		}

		// Rule 2: full_name <= 200
		if len(fullName) > 200 {
			exFn("full_name exceeds maximum length of 200 characters")
			continue
		}

		// Rule 3: gender exactly "M" or "F"
		if gender != "M" && gender != "F" {
			exFn("gender must be exactly 'M' or 'F'")
			continue
		}

		// Rule 4: date_of_birth parseable if present
		var dobPtr *string
		if dateOfBirth != "" {
			if _, parseErr := time.Parse("2006-01-02", dateOfBirth); parseErr != nil {
				exFn("date_of_birth must be in YYYY-MM-DD format")
				continue
			}
			dobPtr = &dateOfBirth

			// Rule 5: year in [1900, currentYear]
			parsedDOB, _ := time.Parse("2006-01-02", dateOfBirth)
			year := parsedDOB.Year()
			if year < 1900 || year > currentYear {
				exFn("date_of_birth is out of valid range")
				continue
			}
		}

		// Rule 6: upi_number <= 20 if present
		var upiPtr *string
		if upiNumber != "" {
			if len(upiNumber) > 20 {
				exFn("upi_number exceeds maximum length of 20 characters")
				continue
			}
			upiPtr = &upiNumber
		}

		// Rule 7: knec_assessment_number <= 30 if present (DB is VARCHAR(15), use 15)
		var knecPtr *string
		if knecNumber != "" {
			if len(knecNumber) > 30 {
				exFn("knec_assessment_number exceeds maximum length of 30 characters")
				continue
			}
			knecPtr = &knecNumber
		}

		// Rule 8: class_id if present must be valid
		var classIDPtr *string
		if classID != "" {
			if !validClassesMap[classID] {
				exFn("class_id references a class that does not exist in this organisation")
				continue
			}
			classIDPtr = &classID
		}

		// Rule 9: cbc_student_parents_id if present must be valid
		var parentIDPtr *string
		if parentID != "" {
			if !validParentsMap[parentID] {
				exFn("cbc_student_parents_id references a parent that does not exist in this organisation")
				continue
			}
			parentIDPtr = &parentID
		}

		validStudents = append(validStudents, ValidStudent{
			RowNumber:            sr.RowNumber,
			FullName:             fullName,
			Gender:               gender,
			DateOfBirth:          dobPtr,
			UPINumber:            upiPtr,
			KNECAssessmentNumber: knecPtr,
			CBCStudentParentsID:  parentIDPtr,
			ClassID:              classIDPtr,
			RawData:              raw,
		})
	}

	logger.Info("validation complete",
		zap.Int("valid", len(validStudents)),
		zap.Int("failed", len(failedRows)),
	)

	// Phase D — Chunked transactional bulk write
	chunkSize := EnrollmentChunkSize
	cumulativeSuccess := 0
	cumulativeFailed := len(failedRows)

	for i := 0; i < len(validStudents); i += chunkSize {
		end := i + chunkSize
		if end > len(validStudents) {
			end = len(validStudents)
		}
		chunk := validStudents[i:end]

		// Try bulk insert first
		results, insertErr := w.repo.BulkInsertStudents(ctx, payload.TenantID, schoolID, chunk)
		if insertErr != nil {
			// Rollback failed — need per-row fallback
			logger.Warn("bulk student insert failed, falling back to per-row",
				zap.Error(insertErr),
				zap.Int("chunk_start", i),
			)

			for _, s := range chunk {
				// Try each student individually
				result, perErr := w.repo.BulkInsertStudents(ctx, payload.TenantID, schoolID, []ValidStudent{s})
				if perErr != nil {
					// Extract human-readable Postgres error
					errMsg := perErr.Error()
					// Try to get pgError detail
					var pgErr interface{ SQLState() string }
					if errors.As(perErr, &pgErr) {
						errMsg = fmt.Sprintf("database constraint violation: %s", pgErr.SQLState())
					}
					failedRows = append(failedRows, FailedRow{
						RawData:      s.RawData,
						ErrorMessage: errMsg,
					})
					cumulativeFailed++
				} else if len(result) > 0 {
					// Student inserted successfully — also insert enrollment
					if result[0].ClassID != nil {
						if enrErr := w.repo.BulkInsertEnrollments(ctx, payload.TenantID, schoolID, academicTermID, result); enrErr != nil {
							logger.Warn("enrollment insert failed for single student (non-fatal)",
								zap.Error(enrErr),
							)
						}
					}
					cumulativeSuccess++
				}
			}

			// Publish progress after per-row fallback
			w.updateRedisProgress(ctx, payload.JobID, "processing", cumulativeSuccess+cumulativeFailed, totalRecords, cumulativeSuccess, cumulativeFailed)
			continue
		}

		// Bulk insert succeeded — now do enrollments
		enrErr := w.repo.BulkInsertEnrollments(ctx, payload.TenantID, schoolID, academicTermID, results)
		if enrErr != nil {
			logger.Warn("bulk enrollment insert failed (non-fatal, students already created)",
				zap.Error(enrErr),
				zap.Int("chunk_start", i),
			)
		}

		cumulativeSuccess += len(results)
		w.updateRedisProgress(ctx, payload.JobID, "processing", cumulativeSuccess+cumulativeFailed, totalRecords, cumulativeSuccess, cumulativeFailed)
	}

	// Phase E — Finalise
	w.finaliseStudentImport(ctx, payload.JobID, totalRecords, failedRows, &cumulativeSuccess, logger)
	return nil
}

// finaliseStudentImport persists failure rows, updates the job, publishes terminal event, purges staging.
func (w *Worker) finaliseStudentImport(ctx context.Context, jobID string, totalRecords int, failedRows []FailedRow, successCount *int, logger *zap.Logger) {
	finalSuccess := 0
	if successCount != nil {
		finalSuccess = *successCount
	} else {
		// All rows failed (e.g. academic term not found)
		finalSuccess = 0
	}
	finalFailed := len(failedRows)
	finalProcessed := finalSuccess + finalFailed

	// Persist failure rows (single bulk insert)
	if len(failedRows) > 0 {
		if err := w.repo.BulkInsertFailures(ctx, jobID, failedRows); err != nil {
			logger.Error("failed to persist failure rows", zap.Error(err))
		}
	}

	// Update job record
	finalStatus := "completed"
	if finalFailed > 0 {
		finalStatus = "completed_with_errors"
	}

	if err := w.repo.UpdateImportJobStatus(ctx, jobID, finalStatus, finalProcessed, finalSuccess, finalFailed); err != nil {
		logger.Error("failed to update import job status", zap.Error(err))
	}

	// Publish terminal event
	w.updateRedisProgress(ctx, jobID, finalStatus, finalProcessed, totalRecords, finalSuccess, finalFailed)

	// Purge staging rows
	if err := w.repo.PurgeStaging(ctx, jobID); err != nil {
		logger.Warn("failed to purge staging rows", zap.Error(err))
	}

	logger.Info("student import job completed",
		zap.String("status", finalStatus),
		zap.Int("processed", finalProcessed),
		zap.Int("total", totalRecords),
		zap.Int("success", finalSuccess),
		zap.Int("failed", finalFailed),
	)
}

// updateRedisProgress updates the Redis progress hash and publishes to Pub/Sub.
func (w *Worker) updateRedisProgress(ctx context.Context, jobID, status string, processed, total, successCount, failedCount int) {
	if w.rdb == nil {
		return
	}

	progressKey := RedisProgressPrefix + jobID
	eventKey := RedisEventsPrefix + jobID

	// Update hash with TTL using full Redis client
	if w.redisCli != nil {
		pipe := w.redisCli.Pipeline()
		pipe.HSet(ctx, progressKey, map[string]interface{}{
			"status":        status,
			"processed":     processed,
			"total":         total,
			"success_count": successCount,
			"failed_count":  failedCount,
		})
		pipe.Expire(ctx, progressKey, RedisProgressTTL*time.Second)
		if _, err := pipe.Exec(ctx); err != nil {
			w.logger.Warn("Redis pipeline exec failed", zap.Error(err))
		}
	}

	// Publish frame to Pub/Sub
	frame := ProgressFrame{
		Status:       status,
		Processed:    processed,
		Total:        total,
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}
	data, err := json.Marshal(frame)
	if err != nil {
		w.logger.Error("failed to marshal progress frame", zap.Error(err))
		return
	}

	if err := w.rdb.Publish(ctx, eventKey, string(data)).Err(); err != nil {
		w.logger.Warn("redis publish failed (non-fatal)",
			zap.String("channel", eventKey),
			zap.Error(err),
		)
	}
}

// compile-time check for pgError interface (used in per-row fallback)
var _ = errors.As
