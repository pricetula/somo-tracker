package timetable

import "go.uber.org/fx"

// Module is an fx-compatible module for the timetable domain.
var Module = fx.Module("timetable",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
