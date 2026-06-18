package cbctimetable

import (
	"go.uber.org/fx"
)

// Module provides the CBC timetable service, repository, and handler.
var Module = fx.Module("cbctimetable",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
