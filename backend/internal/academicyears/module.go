package academicyears

import "go.uber.org/fx"

// Module is an fx-compatible module for the academicyears domain.
var Module = fx.Module("academicyears",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
