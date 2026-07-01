package billing

import "go.uber.org/fx"

// Module is an fx-compatible module for the billing domain.
var Module = fx.Module("billing",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
