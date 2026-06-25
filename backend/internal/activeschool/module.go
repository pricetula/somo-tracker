package activeschool

import "go.uber.org/fx"

// Module is an fx-compatible module for the activeschool domain.
var Module = fx.Module("activeschool",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
