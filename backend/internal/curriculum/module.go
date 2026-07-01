package curriculum

import "go.uber.org/fx"

// Module is an fx-compatible module for the curriculum domain.
var Module = fx.Module("curriculum",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
