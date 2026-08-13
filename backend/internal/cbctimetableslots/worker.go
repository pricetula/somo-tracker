package cbctimetableslots

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// ─── Task Type Constants ──────────────────────────────────────────────────
// These task names match the handlers registered in the attendance worker.
// They are duplicated here intentionally to avoid cross-domain Go imports.

const taskRefreshTeacherWorkloadSummaries = "attendance:refresh_teacher_workload_summaries"

// ─── Task Payload ─────────────────────────────────────────────────────────

type workloadRefreshPayload struct {
	AcademicYearID string `json:"academic_year_id"`
}

// ─── Enqueuer ─────────────────────────────────────────────────────────────

// Enqueuer publishes background workload summary refresh tasks.
type Enqueuer struct {
	client *asynq.Client
	logger *zap.SugaredLogger
}

// NewEnqueuer creates a new Enqueuer. The shared *asynq.Client is injected
// via fx (provided by database.Module) — never construct a client here.
func NewEnqueuer(client *asynq.Client, logger *zap.SugaredLogger) *Enqueuer {
	return &Enqueuer{client: client, logger: logger}
}

// EnqueueWorkloadSummaryRefresh enqueues a task to refresh teacher workload
// summaries for the given academic year. Best-effort: logs failures but
// does not block the caller.
func (e *Enqueuer) EnqueueWorkloadSummaryRefresh(ctx context.Context, academicYearID string) {
	payload, _ := json.Marshal(workloadRefreshPayload{AcademicYearID: academicYearID})
	task := asynq.NewTask(taskRefreshTeacherWorkloadSummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("cbctimetableslots: enqueue workload refresh failed",
			"academic_year_id", academicYearID, "error", err,
		)
	}
}
