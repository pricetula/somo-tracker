package cbcschools

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/curriculum"
)

// ─── Cross-domain seeders (consumer-side adapters) ────────────────────────
//
// These adapters bridge the cbcschools.CurriculumSeeder / AcademicYearSeeder
// consumer interfaces to the concrete implementations in the curriculum and
// academicyears domains. They live here — in the consuming package — so the
// provider packages never need to know about cbcschools (DDD boundary rule).
// The UserSchoolEnroller adapter is provided by the auth module instead:
// auth already imports cbcschools (for SchoolCreator), so providing it here
// would create an import cycle.

// curriculumSeederAdapter adapts *curriculum.SeedingService (which seeds via
// uuid.UUID ids) to the string-based CurriculumSeeder consumer interface, so
// school creation auto-seeds the embedded CBC curriculum.
type curriculumSeederAdapter struct {
	svc *curriculum.SeedingService
}

// SeedForSchool seeds the embedded CBC curriculum for a newly created school.
func (a curriculumSeederAdapter) SeedForSchool(ctx context.Context, tenantID, schoolID string) error {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("cbcschools.curriculumSeederAdapter.SeedForSchool: invalid tenant_id %q: %w", tenantID, err)
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return fmt.Errorf("cbcschools.curriculumSeederAdapter.SeedForSchool: invalid school_id %q: %w", schoolID, err)
	}
	return a.svc.SeedSchoolCurriculumDefault(ctx, tenantUUID, schoolUUID)
}

// newCurriculumSeeder provides the CurriculumSeeder for the fx container.
func newCurriculumSeeder(svc *curriculum.SeedingService) CurriculumSeeder {
	return curriculumSeederAdapter{svc: svc}
}

// newAcademicYearSeeder provides the AcademicYearSeeder for the fx container.
// *academicyears.Service already satisfies the interface directly.
func newAcademicYearSeeder(svc *academicyears.Service) AcademicYearSeeder {
	return svc
}
