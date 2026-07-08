package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"somotracker/backend/internal/database"
)

// Module is an fx-compatible module for the imports engine.
var Module = fx.Module("imports",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(ServiceRepository)),
		),
		NewAsynqClient,
		NewAsynqServer,
		NewService,
		NewHandler,
		NewWorker,
	),
)

// NewAsynqClient creates an Asynq client from the Redis pool.
func NewAsynqClient(pools *database.Pools) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr: pools.Redis.Options().Addr,
	})
}

// NewAsynqServer creates an Asynq server for processing import chunks.
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
// Queues map to assign relative priorities between queues. The "imports"
// queue weight of 10 means it takes all available workers when it is the
// only queue type in the system.
func NewAsynqServer(pools *database.Pools) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr: pools.Redis.Options().Addr,
		},
		asynq.Config{
			Concurrency: 3,
			Queues: map[string]int{
				"imports": 10,
			},
			Logger: asynqLogger{},
		},
	)
}

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

// Worker wraps the Asynq server and task registration.
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewWorker creates a new Worker with chunk processing handler.
func NewWorker(svc *Service, server *asynq.Server) *Worker {
	mux := asynq.NewServeMux()
	mux.HandleFunc("imports:process_chunk", func(ctx context.Context, t *asynq.Task) error {
		var payload ChunkTaskPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("imports:process_chunk: unmarshal payload: %w", err)
		}
		return svc.ProcessChunk(ctx, payload)
	})

	return &Worker{
		server: server,
		mux:    mux,
	}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return fmt.Errorf("imports.Worker.Start: %w", err)
	}
	slog.Info("imports.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	w.server.Shutdown()
	slog.Info("imports.Worker: asynq server stopped")
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

// RedisOptions extracts Redis options from a *redis.Client.
func RedisOptions(rdb *redis.Client) *redis.Options {
	return rdb.Options()
}
