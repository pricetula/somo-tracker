package timetablestructure

import "go.uber.org/fx"

// Module is an fx-compatible module for the timetablestructure domain.
var Module = fx.Module("timetablestructure",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
