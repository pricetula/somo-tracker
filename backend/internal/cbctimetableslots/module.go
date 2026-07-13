package cbctimetableslots

import "go.uber.org/fx"

// Module is an fx-compatible module for the cbctimetableslots domain.
var Module = fx.Module("cbctimetableslots",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
