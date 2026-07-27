package assessments

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

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
	TermID string `json:"term_id"`
}

// ─── Asynq Client ─────────────────────────────────────────────────────────

// NewAsynqClient creates an Asynq client from the Redis pool.
func NewAsynqClient(pools *database.Pools) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr: pools.Redis.Options().Addr,
	})
}

// ─── Enqueuer ─────────────────────────────────────────────────────────────

// Enqueuer publishes background refresh tasks to Asynq.
type Enqueuer struct {
	client *asynq.Client
}

// NewEnqueuer creates a new Enqueuer.
func NewEnqueuer(client *asynq.Client) *Enqueuer {
	return &Enqueuer{client: client}
}

// EnqueueOverallSummaryRefresh enqueues a task to refresh overall summaries.
// Best-effort: logs failures but does not block the caller.
func (e *Enqueuer) EnqueueOverallSummaryRefresh(ctx context.Context, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TermID: termID})
	task := asynq.NewTask(TaskRefreshOverallSummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "assessments: enqueue overall summary refresh failed",
			"term_id", termID, "error", err,
		)
	}
}

// EnqueueProjectionsRefresh enqueues a task to refresh performance projections.
func (e *Enqueuer) EnqueueProjectionsRefresh(ctx context.Context, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TermID: termID})
	task := asynq.NewTask(TaskRefreshProjections, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "assessments: enqueue projections refresh failed",
			"term_id", termID, "error", err,
		)
	}
}

// EnqueueTeacherPerformanceRefresh enqueues a task to refresh teacher summaries.
func (e *Enqueuer) EnqueueTeacherPerformanceRefresh(ctx context.Context, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TermID: termID})
	task := asynq.NewTask(TaskRefreshTeacherPerformance, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "assessments: enqueue teacher perf refresh failed",
			"term_id", termID, "error", err,
		)
	}
}

// EnqueueCohortPositionsRefresh enqueues a task to refresh cohort positions.
func (e *Enqueuer) EnqueueCohortPositionsRefresh(ctx context.Context, termID string) {
	payload, _ := json.Marshal(RefreshTermPayload{TermID: termID})
	task := asynq.NewTask(TaskRefreshCohortPositions, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "assessments: enqueue cohort positions refresh failed",
			"term_id", termID, "error", err,
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
}

// NewWorker creates a new background summary refresh worker.
func NewWorker(pools *database.Pools) *Worker {
	return &Worker{pools: pools, pool: pools.PG}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	w.server = asynq.NewServer(
		asynq.RedisClientOpt{Addr: w.pools.Redis.Options().Addr},
		asynq.Config{
			Concurrency: 2,
			Queues:      map[string]int{"summaries": 10},
			Logger:      asynqLogger{},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskRefreshOverallSummaries, w.handleRefreshOverallSummaries)
	mux.HandleFunc(TaskRefreshProjections, w.handleRefreshProjections)
	mux.HandleFunc(TaskRefreshTeacherPerformance, w.handleRefreshTeacherPerformance)
	mux.HandleFunc(TaskRefreshCohortPositions, w.handleRefreshCohortPositions)

	if err := w.server.Start(mux); err != nil {
		return fmt.Errorf("assessments.Worker.Start: %w", err)
	}
	slog.InfoContext(ctx, "assessments.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	if w.server != nil {
		w.server.Shutdown()
	}
	slog.InfoContext(ctx, "assessments.Worker: asynq server stopped")
	return nil
}

// ─── Task Handlers ────────────────────────────────────────────────────────

func (w *Worker) handleRefreshOverallSummaries(ctx context.Context, t *asynq.Task) error {
	var p RefreshTermPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshOverallSummaries: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "assessments: refreshing overall summaries", "term_id", p.TermID)

	_, err := w.pool.Exec(ctx, `SELECT fn_compute_term_overall_summaries_for_term($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshOverallSummaries: %w", err)
	}
	slog.InfoContext(ctx, "assessments: overall summaries refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleRefreshProjections(ctx context.Context, t *asynq.Task) error {
	var p RefreshTermPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshProjections: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "assessments: refreshing projections", "term_id", p.TermID)

	_, err := w.pool.Exec(ctx, `SELECT fn_compute_performance_projections_for_term($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshProjections: %w", err)
	}
	slog.InfoContext(ctx, "assessments: projections refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleRefreshTeacherPerformance(ctx context.Context, t *asynq.Task) error {
	var p RefreshTermPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshTeacherPerformance: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "assessments: refreshing teacher performance", "term_id", p.TermID)

	_, err := w.pool.Exec(ctx, `SELECT fn_compute_teacher_subject_performance_summaries($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshTeacherPerformance: %w", err)
	}
	slog.InfoContext(ctx, "assessments: teacher performance refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleRefreshCohortPositions(ctx context.Context, t *asynq.Task) error {
	var p RefreshTermPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshCohortPositions: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "assessments: refreshing cohort positions", "term_id", p.TermID)

	_, err := w.pool.Exec(ctx, `SELECT fn_compute_cohort_positions_for_term($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("assessments.Worker.handleRefreshCohortPositions: %w", err)
	}
	slog.InfoContext(ctx, "assessments: cohort positions refreshed",
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

// asynqLogger implements asynq.Logger via slog.
type asynqLogger struct{}

func (asynqLogger) Debug(args ...interface{}) { slog.Debug(fmt.Sprint(args...)) }
func (asynqLogger) Info(args ...interface{})  { slog.Info(fmt.Sprint(args...)) }
func (asynqLogger) Warn(args ...interface{})  { slog.Warn(fmt.Sprint(args...)) }
func (asynqLogger) Error(args ...interface{}) { slog.Error(fmt.Sprint(args...)) }
func (asynqLogger) Fatal(args ...interface{}) { slog.Error(fmt.Sprint(args...)) }
