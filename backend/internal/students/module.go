package students

import "go.uber.org/fx"

// Module is an fx-compatible module for the students domain.
var Module = fx.Module("students",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(StudentRepository))),
		NewService,
		NewHandler,
	),
)
