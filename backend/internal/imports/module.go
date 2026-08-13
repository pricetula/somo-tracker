package imports

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// Module is an fx-compatible module for the imports engine.
// *asynq.Client is provided once by database.Module; the Asynq server and
// scheduler are built via database.NewAsynqServer / database.NewAsynqScheduler
// so the Redis connection options and zap log adapter live in one place.
var Module = fx.Module("imports",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(ServiceRepository)),
		),
		NewCleanupScheduler,
		NewService,
		NewHandler,
		NewWorker,
	),
)

// importsServerConfig is the Asynq server config for import chunk processing.
//
// Concurrency is capped at 3 to leave headroom for other background job
// types (e.g., notifications, exports) that may share the same Asynq server.
// At ChunkSize=100, 3 concurrent workers can process up to 300 rows at a
// time — this is sufficient for MaxImportRows (5000) even with slow per-row
// inserts, while preventing a single large import from starving unrelated
// background work or other tenants' smaller imports enqueued at the same
// time.
//
// When additional queue types are added, increase Concurrency and use the
// Queues map to assign relative priorities. The "imports" queue weight of
// 10 means it takes all available workers when it is the only queue type
// in the system.
var importsServerConfig = asynq.Config{
	Concurrency: 3,
	Queues: map[string]int{
		"imports": 10,
	},
}

// NewCleanupScheduler creates a periodic task scheduler for the retention
// cleanup job. It enqueues imports:cleanup_old_data once per day.
func NewCleanupScheduler(pools *database.Pools, svc *Service, logger *zap.SugaredLogger) *CleanupScheduler {
	return &CleanupScheduler{
		pools:  pools,
		svc:    svc,
		logger: logger,
	}
}

// CleanupScheduler manages the periodic retention cleanup of old import
// staging and failure data. Uses Asynq's Scheduler to register a daily
// recurring task.
type CleanupScheduler struct {
	pools  *database.Pools
	svc    *Service
	logger *zap.SugaredLogger
}

// Start starts the periodic cleanup scheduler. Called via fx lifecycle.
func (cs *CleanupScheduler) Start(ctx context.Context) error {
	scheduler := database.NewAsynqScheduler(cs.pools, cs.logger, nil)

	// Schedule the daily cleanup at 03:00 UTC (low-traffic window).
	task := asynq.NewTask("imports:cleanup_old_data", nil)
	entryID, err := scheduler.Register("@daily", task)
	if err != nil {
		return fmt.Errorf("imports.CleanupScheduler: register cleanup task: %w", err)
	}

	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("imports.CleanupScheduler: start: %w", err)
	}

	cs.logger.Infow("imports.CleanupScheduler: cleanup task registered",
		"entry_id", entryID,
		"schedule", "@daily",
	)
	return nil
}

// Stop stops the cleanup scheduler.
func (cs *CleanupScheduler) Stop(ctx context.Context) error {
	cs.logger.Infow("imports.CleanupScheduler: stopped")
	return nil
}

// Worker wraps the Asynq server and task registration.
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	logger *zap.SugaredLogger
}

// NewWorker creates a new Worker with chunk processing and cleanup handlers.
func NewWorker(svc *Service, pools *database.Pools, logger *zap.SugaredLogger) *Worker {
	server := database.NewAsynqServer(pools, logger, importsServerConfig)

	mux := asynq.NewServeMux()
	mux.HandleFunc("imports:process_chunk", func(ctx context.Context, t *asynq.Task) error {
		var payload ChunkTaskPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("imports:process_chunk: unmarshal payload: %w", err)
		}
		return svc.ProcessChunk(ctx, payload)
	})
	mux.HandleFunc("imports:cleanup_old_data", func(ctx context.Context, t *asynq.Task) error {
		return svc.CleanupExpiredData(ctx)
	})

	return &Worker{
		server: server,
		mux:    mux,
		logger: logger,
	}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return fmt.Errorf("imports.Worker.Start: %w", err)
	}
	w.logger.Info("imports.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	w.server.Shutdown()
	w.logger.Info("imports.Worker: asynq server stopped")
	return nil
}

// RegisterWorkerHooks registers lifecycle hooks for the Asynq worker.
// This should be called from main.go via fx.Invoke.
func RegisterWorkerHooks(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: worker.Start,
		OnStop:  worker.Stop,
	})
}

// RegisterCleanupSchedulerHooks registers lifecycle hooks for the cleanup scheduler.
func RegisterCleanupSchedulerHooks(lc fx.Lifecycle, cs *CleanupScheduler) {
	lc.Append(fx.Hook{
		OnStart: cs.Start,
		OnStop:  cs.Stop,
	})
}
