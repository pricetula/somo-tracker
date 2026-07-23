package classteachers

import "go.uber.org/fx"

// Module is an fx-compatible module for the classteachers domain.
var Module = fx.Module("classteachers",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
