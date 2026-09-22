package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type StudentImportProcessor struct {
	svc    services.StudentImportService
	logger *zap.Logger
	redis  RedisPublisher
}

type StudentBatchPayload struct {
	JobID      string   `json:"job_id"`
	BatchIndex int      `json:"batch_index"`
	ItemIDs    []string `json:"item_ids"`
}

type StudentItemPayload struct {
	AdmissionNumber string                 `json:"admission_number"`
	FullName        string                 `json:"full_name"`
	DateOfBirth     string                 `json:"date_of_birth"`
	Gender          string                 `json:"gender"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func NewStudentImportProcessor(svc services.StudentImportService, logger *zap.Logger, pub RedisPublisher) *StudentImportProcessor {
	return &StudentImportProcessor{
		svc:    svc,
		logger: logger.With(zap.String("worker", "student_import")),
		redis:  pub,
	}
}

func (p *StudentImportProcessor) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload StudentBatchPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("bad job_id: %w", err)
	}

	p.logger.Info("processing student batch", zap.String("job_id", payload.JobID), zap.Int("batch_index", payload.BatchIndex), zap.Int("item_count", len(payload.ItemIDs)))

	_ = p.svc.UpdateJobStatus(ctx, jobID, "PROCESSING")

	const workerConcurrency = 10
	sem := make(chan struct{}, workerConcurrency)
	var wg sync.WaitGroup
	for _, itemIDStr := range payload.ItemIDs {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			p.logger.Warn("bad item id", zap.String("item_id", itemIDStr))
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			p.processSingleItem(ctx, jobID, itemID)
		}()
	}
	wg.Wait()

	p.deriveJobStatus(ctx, jobID)
	return nil
}

func (p *StudentImportProcessor) processSingleItem(ctx context.Context, jobID, itemID uuid.UUID) {
	item, err := p.svc.GetItemByID(ctx, itemID)
	if err != nil {
		p.logger.Error("failed to get item", zap.Error(err), zap.String("item_id", itemID.String()))
		return
	}
	if item.Status != "PENDING" && item.Status != "DEFERRED" {
		return
	}
	attempts := item.AttemptCount + 1
	if err := p.svc.UpdateItemStatus(ctx, itemID, "PROCESSING", nil, "", attempts); err != nil {
		p.logger.Error("failed to mark processing", zap.Error(err))
		return
	}

	var payload StudentItemPayload
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "invalid payload", attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0)
		return
	}

	// Normalize gender
	gender := services.NormalizeGender(payload.Gender)

	// Parse date
	dob, err := services.ParseDate(payload.DateOfBirth)
	if err != nil {
		p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "invalid date", attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0)
		return
	}

	job, err := p.svc.GetJob(ctx, jobID)
	if err != nil || job == nil {
		p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "job not found", attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0)
		return
	}

	metadataBytes, _ := json.Marshal(payload.Metadata)

	err = p.svc.InsertStudent(ctx, job.SchoolID, payload.AdmissionNumber, payload.FullName, dob, gender, metadataBytes)
	if err != nil {
		// Check for unique violation
		if isUniqueViolation(err) {
			p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "admission_number already exists", attempts)
			_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0)
			return
		}
		// Transient? Mark deferred
		p.svc.UpdateItemStatus(ctx, itemID, "DEFERRED", nil, err.Error(), attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 0, 1)
		return
	}

	result, _ := json.Marshal(map[string]string{"admission_number": payload.AdmissionNumber})
	_ = p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", result, "", attempts)
	_ = p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0)
}

func isUniqueViolation(err error) bool {
	// Simple heuristic: Postgres unique violation contains "duplicate key"
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

func (p *StudentImportProcessor) deriveJobStatus(ctx context.Context, jobID uuid.UUID) {
	finalized, err := p.svc.TryFinalizeJob(ctx, jobID)
	if err != nil {
		p.logger.Error("try finalize job failed", zap.Error(err), zap.String("job_id", jobID.String()))
	}
	if finalized {
		p.logger.Info("job finalized", zap.String("job_id", jobID.String()))
	}
	p.publishProgress(ctx, jobID)
}

func (p *StudentImportProcessor) publishProgress(ctx context.Context, jobID uuid.UUID) {
	if p.redis == nil {
		return
	}
	job, err := p.svc.GetJob(ctx, jobID)
	if err != nil || job == nil {
		return
	}
	data, _ := json.Marshal(map[string]interface{}{
		"status":    job.Status,
		"succeeded": job.SucceededCount,
		"failed":    job.FailedCount,
		"deferred":  job.DeferredCount,
		"total":     job.TotalRecords,
	})
	_ = p.redis.Publish(ctx, "bulk_progress_"+jobID.String(), string(data)).Err()
}
