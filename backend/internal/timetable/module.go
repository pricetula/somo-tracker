package timetable

import (
	"go.uber.org/fx"

	"somotracker/backend/internal/academicyears"
)

var Module = fx.Module("timetable",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		fx.Annotate(NewService, fx.As(new(Service))),
		NewHandler,
	),
	fx.Invoke(func(h *Handler, aySvc *academicyears.Service) {
		h.SetAcademicYearsService(aySvc)
	}),
)
