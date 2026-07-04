package cbcschools

import "go.uber.org/fx"

// newServiceWithOptions constructs a Service with optional dependencies.
// This wrapper exists so fx can inject optional CurriculumSeeder and
// UserSchoolEnroller providers without changing NewService's signature.
//
// Both seeder and enroller are optional — if nil, the service operates
// without those capabilities (no automatic curriculum seeding, no
// auto-enrollment of the school creator).
func newServiceWithOptions(
	repo Repository,
	seeder CurriculumSeeder,
	enroller UserSchoolEnroller,
) *Service {
	opts := []ServiceOption{}
	if seeder != nil {
		opts = append(opts, WithCurriculumSeeder(seeder))
	}
	if enroller != nil {
		opts = append(opts, WithUserSchoolEnroller(enroller))
	}
	return NewService(repo, opts...)
}

// Module is an fx-compatible module for the cbcschools domain.
var Module = fx.Module("cbcschools",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		fx.Annotate(
			newServiceWithOptions,
			// Mark CurriculumSeeder and UserSchoolEnroller as optional —
			// if the container doesn't have them, fx passes nil.
			fx.ParamTags(``, `optional:"true"`, `optional:"true"`),
		),
		NewHandler,
	),
)
