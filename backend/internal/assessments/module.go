package assessments

import (
	"context"
	"errors"

	"go.uber.org/fx"

	"somotracker/backend/internal/cbcclasses"
)

// cbcclassesRosterAdapter adapts *cbcclasses.Service to the assessments
// RosterProvider interface, translating cbcclasses sentinels into
// assessments' own sentinels so the service layer never leaks a cross-domain
// error type.
type cbcclassesRosterAdapter struct {
	svc *cbcclasses.Service
}

// GetRosterByClassAndTerm resolves the full class roster for a term via the
// cbcclasses domain and maps it to the assessments RosterStudent shape.
func (a cbcclassesRosterAdapter) GetRosterByClassAndTerm(ctx context.Context, classID, tenantID, schoolID, academicTermID string) ([]RosterStudent, error) {
	result, err := a.svc.GetRoster(ctx, classID, tenantID, schoolID, academicTermID, 1, 1000, "")
	if err != nil {
		if errors.Is(err, cbcclasses.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, cbcclasses.ErrInvalidInput) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}

	students := make([]RosterStudent, 0, len(result.Items))
	for _, entry := range result.Items {
		students = append(students, RosterStudent{
			StudentID:       entry.ID,
			StudentName:     entry.FullName,
			AdmissionNumber: entry.AdmissionNumber,
			Gender:          entry.Gender,
		})
	}
	return students, nil
}

// Module is an fx-compatible module for the assessments domain.
var Module = fx.Module("assessments",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
		// RosterProvider is satisfied by an adapter over *cbcclasses.Service.
		fx.Annotate(
			func(svc *cbcclasses.Service) RosterProvider { return cbcclassesRosterAdapter{svc: svc} },
			fx.As(new(RosterProvider)),
		),
		// Background summary refresh infrastructure
		NewEnqueuer,
		NewWorker,
	),
	// Wire the enqueuer into the service after construction
	fx.Invoke(func(svc *Service, enqueuer *Enqueuer) {
		svc.SetEnqueuer(enqueuer)
	}),
	// Wire the roster provider into the service after construction
	fx.Invoke(func(svc *Service, rp RosterProvider) {
		svc.SetRosterProvider(rp)
	}),
	// Register lifecycle hooks for the background worker
	fx.Invoke(RegisterWorkerHooks),
)
