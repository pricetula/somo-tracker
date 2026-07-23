package cbcstreams

import "go.uber.org/fx"

// Module is an fx-compatible module for the cbcstreams domain.
var Module = fx.Module("cbcstreams",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
