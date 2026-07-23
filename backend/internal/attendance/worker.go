package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"

	"somotracker/backend/internal/database"
)

// ─── Asynq Server ─────────────────────────────────────────────────────────

// newAsynqServer creates a dedicated Asynq server for attendance background
// tasks. Concurrency is set to 1 for summary recomputes because each task
// runs a DB query and we want to avoid overwhelming Postgres during peak
// hours (e.g. 5pm when multiple teachers mark simultaneously).
func newAsynqServer(pools *database.Pools) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr: pools.Redis.Options().Addr,
		},
		asynq.Config{
			Concurrency: 1,
			Queues: map[string]int{
				"attendance": 10,
			},
			Logger: asynqLogger{},
		},
	)
}

// ─── Task Handler ─────────────────────────────────────────────────────────

// recomputeHandler processes attendance:recompute_class_summaries tasks.
type recomputeHandler struct {
	svc   *Service
	dedup Deduplicator
}

// handleClassRecompute recomputes attendance_term_summaries for the
// class specified in the task payload. Deletes the Redis pending flag
// before running so subsequent marks for the same class can enqueue a
// new task immediately.
func (h *recomputeHandler) handleClassRecompute(ctx context.Context, t *asynq.Task) error {
	var payload recomputeClassPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("attendance.recomputeHandler: unmarshal: %w", err)
	}

	// Clear the pending flag FIRST. If we crash after clearing but before
	// the DB write, the next mark will enqueue a fresh task. This is
	// preferable to leaving the flag alive and silently dropping future
	// recomputes.
	dedupKey := fmt.Sprintf("attendance:pending:%s:%s", payload.TermID, payload.ClassID)
	if err := h.dedup.Del(ctx, dedupKey); err != nil {
		slog.WarnContext(ctx, "attendance.recomputeHandler: clear pending flag failed",
			slog.String("error", err.Error()),
			slog.String("dedup_key", dedupKey),
		)
	}

	start := time.Now()
	count, err := h.svc.ComputeClassSummaries(ctx, payload.TenantID, payload.SchoolID, payload.TermID, payload.ClassID)
	if err != nil {
		return fmt.Errorf("attendance.recomputeHandler: compute: %w", err)
	}

	slog.InfoContext(ctx, "attendance.recomputeHandler: summaries recomputed",
		slog.String("tenant_id", payload.TenantID),
		slog.String("school_id", payload.SchoolID),
		slog.String("term_id", payload.TermID),
		slog.String("class_id", payload.ClassID),
		slog.Int("count", count),
		slog.Duration("elapsed", time.Since(start)),
	)
	return nil
}

// ─── Worker ───────────────────────────────────────────────────────────────

// Worker wraps the Asynq server and task routing for attendance tasks.
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewWorker creates a new Worker with the class recompute handler registered.
func NewWorker(svc *Service, dedup Deduplicator, pools *database.Pools) *Worker {
	server := newAsynqServer(pools)
	h := &recomputeHandler{svc: svc, dedup: dedup}
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeRecomputeClassSummaries, h.handleClassRecompute)
	return &Worker{
		server: server,
		mux:    mux,
	}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return fmt.Errorf("attendance.Worker.Start: %w", err)
	}
	slog.Info("attendance.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	w.server.Shutdown()
	slog.Info("attendance.Worker: asynq server stopped")
	return nil
}

// RegisterWorkerHooks registers lifecycle hooks for the attendance Asynq worker.
// This should be called from main.go via fx.Invoke.
func RegisterWorkerHooks(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: worker.Start,
		OnStop:  worker.Stop,
	})
}

// ─── Logger ───────────────────────────────────────────────────────────────

// asynqLogger wraps slog to implement asynq.Logger.
type asynqLogger struct{}

func (l asynqLogger) Debug(args ...interface{}) {
	slog.Debug(fmt.Sprint(args...))
}

func (l asynqLogger) Info(args ...interface{}) {
	slog.Info(fmt.Sprint(args...))
}

func (l asynqLogger) Warn(args ...interface{}) {
	slog.Warn(fmt.Sprint(args...))
}

func (l asynqLogger) Error(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
}

func (l asynqLogger) Fatal(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
}
