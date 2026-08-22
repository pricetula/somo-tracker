package attendance

import (
	"go.uber.org/fx"

	"somotracker/backend/internal/academicyears"
)

// Module is an fx-compatible module for the attendance domain.
var Module = fx.Module("attendance",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		// Background summary refresh infrastructure
		NewEnqueuer,
		NewWorker,
	),
	// Wire the enqueuer into the service after construction
	fx.Invoke(func(svc *Service, enqueuer *Enqueuer) {
		svc.SetEnqueuer(enqueuer)
	}),
	// Wire the enqueuer into the worker so upstream refresh handlers can
	// chain dependent rollup tasks (class_learning_area_term_summaries,
	// class_term_attendance_summaries) after their source tables refresh.
	fx.Invoke(func(w *Worker, enqueuer *Enqueuer) {
		w.SetEnqueuer(enqueuer)
	}),
	// Register lifecycle hooks for the background worker
	fx.Invoke(RegisterWorkerHooks),
	// Wire academicyears service into the handler
	fx.Invoke(func(h *Handler, aySvc *academicyears.Service) {
		h.SetAcademicYearsService(aySvc)
	}),
)
