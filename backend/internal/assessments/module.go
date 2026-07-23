package assessments

import "go.uber.org/fx"

// Module is an fx-compatible module for the assessments domain.
var Module = fx.Module("assessments",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
