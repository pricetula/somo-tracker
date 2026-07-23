package health

import "go.uber.org/fx"

// Module is an fx-compatible module for the health domain.
var Module = fx.Module("health",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
