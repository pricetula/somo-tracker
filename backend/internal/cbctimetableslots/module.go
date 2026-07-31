package cbctimetableslots

import "go.uber.org/fx"

// Module is an fx-compatible module for the cbctimetableslots domain.
var Module = fx.Module("cbctimetableslots",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		// Background workload summary refresh
		NewEnqueuer,
	),
	// Wire the enqueuer into the service after construction
	fx.Invoke(func(svc *Service, enqueuer *Enqueuer) {
		svc.SetEnqueuer(enqueuer)
	}),
)
