package store

import (
	"context"

	"github.com/google/uuid"
	storegen "somotracker/internal/auth/store/gen"
)

// StoreInterface defines the methods the education service needs from the store.
// *Store satisfies this interface automatically.
type StoreInterface interface {
	ListCountries(ctx context.Context) ([]storegen.Country, error)
	GetCountry(ctx context.Context, id uuid.UUID) (storegen.Country, error)
	ListEducationSystems(ctx context.Context) ([]storegen.ListEducationSystemsRow, error)
	GetEducationSystem(ctx context.Context, id uuid.UUID) (storegen.GetEducationSystemRow, error)
	ListEducationSystemLevels(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemLevelsRow, error)
	ListEducationSystemGradingBands(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemGradingBandsRow, error)
	ListEducationSystemSubjects(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemSubjectsRow, error)
}
