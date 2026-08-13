package auth

import (
	"context"
	"errors"

	"go.uber.org/fx"

	"somotracker/backend/internal/cbcschools"
)

// cbcschoolServiceAdapter adapts *cbcschools.Service to the auth SchoolCreator
// interface, translating cbcschools sentinels into auth's own sentinels so the
// auth service layer never leaks a cross-domain error type.
type cbcschoolServiceAdapter struct {
	svc *cbcschools.Service
}

func (a cbcschoolServiceAdapter) CreateSchool(ctx context.Context, tenantID string, name string, role string, creatorUserID ...string) (string, error) {
	return a.svc.CreateSchool(ctx, tenantID, name, role, creatorUserID...)
}

func (a cbcschoolServiceAdapter) GetSchoolByName(ctx context.Context, tenantID, name string) (string, error) {
	id, err := a.svc.GetSchoolByName(ctx, tenantID, name)
	if err != nil {
		if errors.Is(err, cbcschools.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	if id == nil {
		return "", ErrNotFound
	}
	return id.ID, nil
}

// Module is an fx-compatible module for the auth domain (requirement 15).
// It provides all auth dependencies: IdentityProvider (StytchAdapter),
// Repository (SqlcRepository), Service, and Handler.
//
// Config is expected to be provided by the application root via config.Module.
// *database.Pools is provided by database.Module.
// *zap.Logger is expected to be provided by the application root.
// *http.Client is provided by utils.Module.
// SchoolCreator is provided by cbcschools.Module.
var Module = fx.Module("auth",
	fx.Provide(
		// 1. Interfaces use fx.Annotate + fx.As
		fx.Annotate(
			NewStytchAdapter,
			fx.As(new(IdentityProvider)),
		),
		fx.Annotate(
			NewSqlcRepository,
			fx.As(new(Repository)),
			// Also expose the repository as cbcschools.UserSchoolEnroller so
			// school creation auto-enrolls the creator as SCHOOL_ADMIN and
			// sets their active school. cbcschools cannot provide this itself
			// — it would create an import cycle (auth → cbcschools).
			fx.As(new(cbcschools.UserSchoolEnroller)),
		),
		// 2. SchoolCreator from cbcschools.Service (error-translating adapter)
		fx.Annotate(
			func(svc *cbcschools.Service) SchoolCreator { return cbcschoolServiceAdapter{svc: svc} },
			fx.As(new(SchoolCreator)),
		),

		// 3. Concrete Structs (Service & Handler) are provided directly
		NewService,
		NewHandler,
	),
)
