package auth

import "go.uber.org/fx"

// Module is an fx-compatible module for the auth domain (requirement 15).
// It provides all auth dependencies: IdentityProvider (StytchAdapter),
// Repository (SqlcRepository), Service, and Handler.
//
// Config is expected to be provided by the application root via config.Module.
// *database.Pools is provided by database.Module.
// *zap.Logger is expected to be provided by the application root.
// *http.Client is provided by utils.Module.
var Module = fx.Module("auth",
	fx.Provide(
		fx.Annotate(
			NewStytchAdapter,
			fx.As(new(IdentityProvider)),
		),
		fx.Annotate(
			NewSqlcRepository,
			fx.As(new(Repository)),
		),
		NewService,
		NewHandler,
	),
)
