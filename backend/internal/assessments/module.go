package assessments

import "go.uber.org/fx"

// Module is an fx-compatible module for the assessments domain.
var Module = fx.Module("assessments",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		// Background summary refresh infrastructure
		NewAsynqClient,
		NewEnqueuer,
		NewWorker,
	),
	// Wire the enqueuer into the service after construction
	fx.Invoke(func(svc *Service, enqueuer *Enqueuer) {
		svc.SetEnqueuer(enqueuer)
	}),
	// Register lifecycle hooks for the background worker
	fx.Invoke(RegisterWorkerHooks),
)
