package teacherperformance

import "go.uber.org/fx"

// Module is an fx-compatible module for the teacherperformance domain.
var Module = fx.Module("teacherperformance",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
