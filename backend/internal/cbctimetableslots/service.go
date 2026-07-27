package cbctimetableslots

import (
	"context"
	"fmt"
)

// Service contains business logic for the cbctimetableslots domain.
type Service struct {
	Repo     Repository
	enqueuer *Enqueuer
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// SetEnqueuer sets the background task enqueuer for workload summary refreshes.
func (s *Service) SetEnqueuer(e *Enqueuer) {
	s.enqueuer = e
}

// ListSlots returns slots matching the filter.
func (s *Service) ListSlots(ctx context.Context, filter SlotFilter) (*SlotListResult, error) {
	if filter.AcademicYearID == "" {
		return nil, fmt.Errorf("cbctimetableslots.Service.ListSlots: %w", ErrInvalidInput)
	}

	slots, err := s.Repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.ListSlots: %w", err)
	}

	return &SlotListResult{Items: slots, Total: len(slots)}, nil
}

// ListEnrichedSlots returns slots with joined data for the scheduling board.
func (s *Service) ListEnrichedSlots(ctx context.Context, filter SlotFilter) (*EnrichedSlotListResult, error) {
	if filter.AcademicYearID == "" {
		return nil, fmt.Errorf("cbctimetableslots.Service.ListEnrichedSlots: %w", ErrInvalidInput)
	}

	slots, err := s.Repo.ListEnriched(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.ListEnrichedSlots: %w", err)
	}

	return &EnrichedSlotListResult{Items: slots, Total: len(slots)}, nil
}

// GetSlot returns a single slot by ID.
func (s *Service) GetSlot(ctx context.Context, id string) (*SlotWithEnrichedData, error) {
	if id == "" {
		return nil, fmt.Errorf("cbctimetableslots.Service.GetSlot: %w", ErrInvalidInput)
	}

	slot, err := s.Repo.GetEnrichedByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.GetSlot: %w", err)
	}

	return slot, nil
}

// CreateSlot creates a single slot assignment with tenant and school scoping.
func (s *Service) CreateSlot(ctx context.Context, tenantID, schoolID string, payload CreateSlotPayload) (*TimetableSlot, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbctimetableslots.Service.CreateSlot: %w", ErrInvalidInput)
	}

	if err := validateCreatePayload(payload); err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.CreateSlot: %w", err)
	}

	slot, err := s.Repo.Create(ctx, tenantID, schoolID, payload)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.CreateSlot: %w", err)
	}

	// Asynchronously refresh teacher workload summaries for the year.
	if s.enqueuer != nil && payload.AcademicYearID != "" {
		s.enqueuer.EnqueueWorkloadSummaryRefresh(ctx, payload.AcademicYearID)
	}

	return slot, nil
}

// BatchCreateSlots creates multiple slots atomically with tenant and school scoping.
func (s *Service) BatchCreateSlots(ctx context.Context, tenantID, schoolID string, payload BatchCreateSlotsPayload) (*SlotListResult, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbctimetableslots.Service.BatchCreateSlots: %w", ErrInvalidInput)
	}

	if len(payload.Slots) == 0 {
		return nil, fmt.Errorf("cbctimetableslots.Service.BatchCreateSlots: at least one slot required: %w", ErrInvalidInput)
	}

	for i, sl := range payload.Slots {
		if err := validateCreatePayload(sl); err != nil {
			return nil, fmt.Errorf("cbctimetableslots.Service.BatchCreateSlots: slot %d: %w", i, err)
		}
	}

	slots, err := s.Repo.BatchCreate(ctx, tenantID, schoolID, payload.Slots)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.BatchCreateSlots: %w", err)
	}

	// Asynchronously refresh teacher workload summaries for the year.
	if s.enqueuer != nil && payload.Slots[0].AcademicYearID != "" {
		s.enqueuer.EnqueueWorkloadSummaryRefresh(ctx, payload.Slots[0].AcademicYearID)
	}

	return &SlotListResult{Items: slots, Total: len(slots)}, nil
}

// UpdateSlot updates an existing slot assignment.
func (s *Service) UpdateSlot(ctx context.Context, id string, payload UpdateSlotPayload) (*TimetableSlot, error) {
	if id == "" {
		return nil, fmt.Errorf("cbctimetableslots.Service.UpdateSlot: %w", ErrInvalidInput)
	}

	slot, err := s.Repo.Update(ctx, id, payload)
	if err != nil {
		return nil, fmt.Errorf("cbctimetableslots.Service.UpdateSlot: %w", err)
	}

	// Asynchronously refresh teacher workload summaries for the year.
	if s.enqueuer != nil && slot != nil && slot.AcademicYearID != "" {
		s.enqueuer.EnqueueWorkloadSummaryRefresh(ctx, slot.AcademicYearID)
	}

	return slot, nil
}

// DeleteSlot deletes a slot by ID.
func (s *Service) DeleteSlot(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("cbctimetableslots.Service.DeleteSlot: %w", ErrInvalidInput)
	}

	return s.Repo.Delete(ctx, id)
}

// ClearDay removes all slots for given structure IDs.
func (s *Service) ClearDay(ctx context.Context, structureIDs []string) error {
	if len(structureIDs) == 0 {
		return fmt.Errorf("cbctimetableslots.Service.ClearDay: %w", ErrInvalidInput)
	}

	return s.Repo.ClearDay(ctx, structureIDs)
}

// ClearClassDay removes all slots for a specific class on a structure day.
func (s *Service) ClearClassDay(ctx context.Context, structureID, classID string) error {
	if structureID == "" || classID == "" {
		return fmt.Errorf("cbctimetableslots.Service.ClearClassDay: %w", ErrInvalidInput)
	}

	return s.Repo.ClearClassDay(ctx, structureID, classID)
}

// validateCreatePayload validates a slot creation request.
func validateCreatePayload(payload CreateSlotPayload) error {
	if payload.AcademicYearID == "" {
		return fmt.Errorf("%w: academic_year_id is required", ErrInvalidInput)
	}
	if payload.StructureID == "" {
		return fmt.Errorf("%w: structure_id is required", ErrInvalidInput)
	}
	if payload.ClassID == "" {
		return fmt.Errorf("%w: class_id is required", ErrInvalidInput)
	}
	if payload.LearningAreaID == "" {
		return fmt.Errorf("%w: learning_area_id is required", ErrInvalidInput)
	}
	if payload.TeacherID == "" {
		return fmt.Errorf("%w: teacher_id is required", ErrInvalidInput)
	}
	return nil
}
