package cohortpositions

import "go.uber.org/fx"

// Module is an fx-compatible module for the cohort positions domain.
// Note: *asynq.Server and *asynq.Client are already provided by the imports
// module. The cohortpositions Worker and Scheduler create their own asynq
// instances internally to avoid fx duplicate-provider conflicts.
var Module = fx.Module("cohortpositions",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		NewWorker,
		NewRefreshScheduler,
	),
)
