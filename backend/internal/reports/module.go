package reports

import "go.uber.org/fx"

// Module is an fx-compatible module for the reports domain.
var Module = fx.Module("reports",
	fx.Provide(
		NewService,
		NewHandler,
	),
)
