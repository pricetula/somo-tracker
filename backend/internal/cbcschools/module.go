package cbcschools

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// newServiceWithOptions constructs a Service with optional dependencies.
// This wrapper exists so fx can inject optional CurriculumSeeder,
// AcademicYearSeeder, and UserSchoolEnroller providers without changing
// NewService's signature.
//
// All three are optional — if nil, the service operates without those
// capabilities (no automatic curriculum seeding, no academic year setup,
// no auto-enrollment of the school creator).
func newServiceWithOptions(
	repo Repository,
	logger *zap.SugaredLogger,
	seeder CurriculumSeeder,
	yearSeeder AcademicYearSeeder,
	enroller UserSchoolEnroller,
) *Service {
	opts := []ServiceOption{}
	if seeder != nil {
		opts = append(opts, WithCurriculumSeeder(seeder))
	}
	if yearSeeder != nil {
		opts = append(opts, WithAcademicYearSeeder(yearSeeder))
	}
	if enroller != nil {
		opts = append(opts, WithUserSchoolEnroller(enroller))
	}
	return NewService(repo, logger, opts...)
}

// Module is an fx-compatible module for the cbcschools domain.
var Module = fx.Module("cbcschools",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		fx.Annotate(
			newServiceWithOptions,
			// Mark CurriculumSeeder, AcademicYearSeeder, and
			// UserSchoolEnroller as optional — if the container doesn't
			// have them, fx passes nil.
			fx.ParamTags(``, `optional:"true"`, `optional:"true"`, `optional:"true"`),
		),
		NewHandler,
	),
)
