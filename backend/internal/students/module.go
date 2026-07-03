package students

import (
	"go.uber.org/fx"

	"somotracker/backend/internal/imports"
)

// Module is an fx-compatible module for the students domain.
var Module = fx.Module("students",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(StudentRepository)),
			fx.As(new(ImportRepository)),
		),
		NewService,
		NewHandler,
		NewStudentImporter,
	),
	// Wire import repository into the service
	fx.Invoke(func(svc *Service, repo ImportRepository) {
		svc.SetImportRepo(repo)
	}),
	// Wire import service adapter into the handler
	fx.Invoke(func(h *Handler, impSvc *imports.Service) {
		h.SetImportService(impSvc)
	}),
	// Register the student Importer with the imports engine at startup
	fx.Invoke(registerStudentImporter),
)

// registerStudentImporter registers the student Importer in the global registry.
func registerStudentImporter(si *StudentImporter) {
	imports.RegisterImporter(si)
}
