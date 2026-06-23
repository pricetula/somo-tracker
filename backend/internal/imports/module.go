package imports

import (
	"github.com/hibiken/asynq"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Module is an fx-compatible module for the imports domain.
// Provides Repository, Service, Worker, and Handler.
var Module = fx.Module("imports",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		func(repo Repository, client *asynq.Client, cfg config.Config, logger *zap.Logger) *Service {
			return NewService(repo, client, cfg, logger)
		},
		NewWorker,
		NewHandler,
	),
)

// AsynqModule provides the Asynq client and server as shared singletons.
var AsynqModule = fx.Module("asynq",
	fx.Provide(
		func(pools *database.Pools) *asynq.Client {
			return NewAsynqClient(pools.Redis)
		},
	),
)

// AsynqServerModule provides and starts the Asynq background worker server.
// This is invoked in the main function to keep the worker running.
// The Worker (provided by imports.Module) is injected so its ErrorHandler
// can be registered for dead-letter handling.
var AsynqServerModule = fx.Module("asynq_server",
	fx.Provide(
		func(pools *database.Pools, cfg config.Config, worker *Worker) *asynq.Server {
			return NewAsynqServer(pools.Redis, cfg, asynq.ErrorHandlerFunc(worker.HandleError))
		},
	),
)
