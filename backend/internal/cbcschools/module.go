package cbcschools

import "go.uber.org/fx"

// Module is an fx-compatible module for the cbcschools domain.
var Module = fx.Module("cbcschools",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
	),
)
