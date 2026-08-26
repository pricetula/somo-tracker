package cohortpositions

import (
	"go.uber.org/fx"

	"somotracker/backend/internal/academicyears"
)

// Module is an fx-compatible module for the cohort positions domain.
// Note: *asynq.Client is provided once by database.Module; the cohortpositions
// Worker and Scheduler build their own asynq instances via
// database.NewAsynqServer / database.NewAsynqScheduler with per-domain configs.
var Module = fx.Module("cohortpositions",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		NewWorker,
		NewRefreshScheduler,
	),
	// Wire academicyears service into the service
	fx.Invoke(func(svc *Service, aySvc *academicyears.Service) {
		svc.SetAcademicYearsService(aySvc)
	}),
)
