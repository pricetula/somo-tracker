package behavior

import "go.uber.org/fx"

// Module is an fx-compatible module for the behavior domain.
var Module = fx.Module("behavior",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
