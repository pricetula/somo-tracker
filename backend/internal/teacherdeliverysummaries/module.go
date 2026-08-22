package teacherdeliverysummaries

import (
	"go.uber.org/fx"

	"somotracker/backend/internal/academicyears"
)

// Module is an fx-compatible module for the teacherdeliverysummaries domain.
var Module = fx.Module("teacherdeliverysummaries",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
	// Wire academicyears service into the handler
	fx.Invoke(func(h *Handler, aySvc *academicyears.Service) {
		h.SetAcademicYearsService(aySvc)
	}),
)
