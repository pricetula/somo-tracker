package teacherworkloadsummaries

import "go.uber.org/fx"

// Module is an fx-compatible module for the teacherworkloadsummaries domain.
var Module = fx.Module("teacherworkloadsummaries",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
