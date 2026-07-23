package attendance

import "go.uber.org/fx"

// Module is an fx-compatible module for the attendance domain.
var Module = fx.Module("attendance",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
