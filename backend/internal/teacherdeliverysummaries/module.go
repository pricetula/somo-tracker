package teacherdeliverysummaries

import "go.uber.org/fx"

// Module is an fx-compatible module for the teacherdeliverysummaries domain.
var Module = fx.Module("teacherdeliverysummaries",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
