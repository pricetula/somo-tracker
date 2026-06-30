package students

import "go.uber.org/fx"

// Module is an fx-compatible module for the students domain.
var Module = fx.Module("students",
	fx.Provide(
		NewRepository,
		func(repo *PgRepository) StudentRepository { return repo },
		NewService,
		NewHandler,
	),
)
