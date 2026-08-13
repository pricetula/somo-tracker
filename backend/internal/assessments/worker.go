package assessments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// ─── Task Type Constants ──────────────────────────────────────────────────

const (
	// TaskRefreshOverallSummaries refreshes student_term_overall_summaries.
	TaskRefreshOverallSummaries = "assessments:refresh_overall_summaries"

	// TaskRefreshProjections refreshes student_performance_projections.
	TaskRefreshProjections = "assessments:refresh_projections"

	// TaskRefreshTeacherPerformance refreshes teacher_subject_performance_summaries.
	TaskRefreshTeacherPerformance = "assessments:refresh_teacher_performance"

	// TaskRefreshCohortPositions refreshes student_cohort_position_summaries.
	TaskRefreshCohortPositions = "assessments:refresh_cohort_positions"
)

// ─── Task Payloads ────────────────────────────────────────────────────────

// RefreshTermPayload is the payload for term-scoped background refresh tasks.
type RefreshTermPayload struct {
	TenantID string `json:"tenant_id"`
	TermID   string `json:"term_id"`
}

// ─── Enqueuer ─────────────────────────────────────────────────────────────

// Enqueuer publishes background refresh tasks to Asynq.
type Enqueuer struct {
	client *asynq.Client
	logger *zap.SugaredLogger
}

// NewEnqueuer creates a new Enqueuer.
func NewEnqueuer(client *asynq.Client, logger *zap.SugaredLogger) *Enqueuer {
	return &Enqueuer{client: client, logger: logger}
}

// EnqueueOverallSummaryRefresh enqueues a task to refresh overall summaries.
// Best-effort: logs failures but does not block the caller.
func (e *Enqueuer) EnqueueOverallSummaryRefresh(ctx context.Context, tenantID, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TenantID: tenantID, TermID: termID})
	task := asynq.NewTask(TaskRefreshOverallSummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("assessments: enqueue overall summary refresh failed",
			"tenant_id", tenantID, "term_id", termID, "error", err,
		)
	}
}

// EnqueueProjectionsRefresh enqueues a task to refresh performance projections.
func (e *Enqueuer) EnqueueProjectionsRefresh(ctx context.Context, tenantID, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TenantID: tenantID, TermID: termID})
	task := asynq.NewTask(TaskRefreshProjections, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("assessments: enqueue projections refresh failed",
			"tenant_id", tenantID, "term_id", termID, "error", err,
		)
	}
}

// EnqueueTeacherPerformanceRefresh enqueues a task to refresh teacher summaries.
func (e *Enqueuer) EnqueueTeacherPerformanceRefresh(ctx context.Context, tenantID, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TenantID: tenantID, TermID: termID})
	task := asynq.NewTask(TaskRefreshTeacherPerformance, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("assessments: enqueue teacher perf refresh failed",
			"tenant_id", tenantID, "term_id", termID, "error", err,
		)
	}
}

// EnqueueCohortPositionsRefresh enqueues a task to refresh cohort positions.
func (e *Enqueuer) EnqueueCohortPositionsRefresh(ctx context.Context, tenantID, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TenantID: tenantID, TermID: termID})
	task := asynq.NewTask(TaskRefreshCohortPositions, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("assessments: enqueue cohort positions refresh failed",
			"tenant_id", tenantID, "term_id", termID, "error", err,
		)
	}
}

// ─── Worker ───────────────────────────────────────────────────────────────

// Worker processes background summary refresh tasks. Uses the PG pool
// directly to call PL/pgSQL functions across domains (no cross-domain Go
// imports), in accordance with the architecture contract.
type Worker struct {
	pools  *database.Pools
	pool   *pgxpool.Pool
	server *asynq.Server
	logger *zap.SugaredLogger
}

// NewWorker creates a new background summary refresh worker.
func NewWorker(pools *database.Pools, logger *zap.SugaredLogger) *Worker {
	return &Worker{pools: pools, pool: pools.PG, logger: logger}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	w.server = database.NewAsynqServer(w.pools, w.logger, asynq.Config{
		Concurrency: 2,
		Queues:      map[string]int{"summaries": 10},
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskRefreshOverallSummaries, w.withTenant(w.handleRefreshOverallSummaries))
	mux.HandleFunc(TaskRefreshProjections, w.withTenant(w.handleRefreshProjections))
	mux.HandleFunc(TaskRefreshTeacherPerformance, w.withTenant(w.handleRefreshTeacherPerformance))
	mux.HandleFunc(TaskRefreshCohortPositions, w.withTenant(w.handleRefreshCohortPositions))

	if err := w.server.Start(mux); err != nil {
		return fmt.Errorf("assessments.Worker.Start: %w", err)
	}
	w.logger.Infow("assessments.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	if w.server != nil {
		w.server.Shutdown()
	}
	w.logger.Infow("assessments.Worker: asynq server stopped")
	return nil
}

// ─── Task Handlers ────────────────────────────────────────────────────────

// withTenant wraps an asynq handler so its task runs with the payload's
// tenant scoped into RLS context (transaction-scoped GUC). Each summary
// refresh function queries RLS-protected tables; without tenant context
// those queries would silently return zero rows.
func (w *Worker) withTenant(h func(ctx context.Context, p RefreshTermPayload) error) func(ctx context.Context, t *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var p RefreshTermPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("assessments.Worker.withTenant: unmarshal: %w", err)
		}
		if p.TenantID == "" {
			return fmt.Errorf("assessments.Worker.withTenant: payload missing tenant_id")
		}
		tctx := database.WithTenantID(ctx, p.TenantID)
		tx, err := database.Begin(tctx, w.pool)
		if err != nil {
			return fmt.Errorf("assessments.Worker.withTenant: begin: %w", err)
		}
		defer func() {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}()
		if err := h(database.WithTenantTx(tctx, tx), p); err != nil {
			return err
		}
		return tx.Commit(context.WithoutCancel(ctx))
	}
}

func (w *Worker) handleRefreshOverallSummaries(ctx context.Context, p RefreshTermPayload) error {
	start := time.Now()
	w.logger.Infow("assessments: refreshing overall summaries", "term_id", p.TermID)

	_, err := database.FromContext(ctx, w.pool).Exec(ctx, `SELECT fn_compute_term_overall_summaries_for_term($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshOverallSummaries: %w", err)
	}
	w.logger.Infow("assessments: overall summaries refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleRefreshProjections(ctx context.Context, p RefreshTermPayload) error {
	start := time.Now()
	w.logger.Infow("assessments: refreshing projections", "term_id", p.TermID)

	_, err := database.FromContext(ctx, w.pool).Exec(ctx, `SELECT fn_compute_performance_projections_for_term($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshProjections: %w", err)
	}
	w.logger.Infow("assessments: projections refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleRefreshTeacherPerformance(ctx context.Context, p RefreshTermPayload) error {
	start := time.Now()
	w.logger.Infow("assessments: refreshing teacher performance", "term_id", p.TermID)

	_, err := database.FromContext(ctx, w.pool).Exec(ctx, `SELECT fn_compute_teacher_subject_performance_summaries($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshTeacherPerformance: %w", err)
	}
	w.logger.Infow("assessments: teacher performance refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleRefreshCohortPositions(ctx context.Context, p RefreshTermPayload) error {
	start := time.Now()
	w.logger.Infow("assessments: refreshing cohort positions", "term_id", p.TermID)

	_, err := database.FromContext(ctx, w.pool).Exec(ctx, `SELECT fn_compute_cohort_positions_for_term($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshCohortPositions: %w", err)
	}
	w.logger.Infow("assessments: cohort positions refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

// ─── Lifecycle Hooks ──────────────────────────────────────────────────────

// RegisterWorkerHooks registers lifecycle hooks for the background worker.
func RegisterWorkerHooks(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: worker.Start,
		OnStop:  worker.Stop,
	})
}
