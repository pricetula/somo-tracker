package reports

import "go.uber.org/fx"

// Module is an fx-compatible module for the reports domain.
var Module = fx.Module("reports",
	fx.Provide(
		NewService,
		NewHandler,
		// Cross-domain provider adapters (providers.go) — the reports module
		// is the joint point that imports the domains it orchestrates.
		newStudentProvider,
		newTermProvider,
		newAttendanceProvider,
		newAssessmentProvider,
		newBehaviorProvider,
	),
)
