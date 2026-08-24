package timetable

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type ServiceImpl struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) *ServiceImpl {
	return &ServiceImpl{repo: repo, logger: logger}
}

// ListBlocks lists all blocks for a tenant/school/year
func (s *ServiceImpl) ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
	blocks, err := s.repo.ListBlocks(ctx, tenantID, schoolID, yearID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.ListBlocks: %w", err)
	}
	return blocks, nil
}

// GetBlock retrieves a single block by ID
func (s *ServiceImpl) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	b, err := s.repo.GetBlock(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.GetBlock: %w", err)
	}
	return b, nil
}

// CreateBlock creates a new time block
func (s *ServiceImpl) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	b, err := s.repo.CreateBlock(ctx, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateBlock: %w", err)
	}
	return b, nil
}

// UpdateBlock updates an existing block
func (s *ServiceImpl) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	b, err := s.repo.UpdateBlock(ctx, id, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.UpdateBlock: %w", err)
	}
	return b, nil
}

// DeleteBlock deletes a block by ID
func (s *ServiceImpl) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	err := s.repo.DeleteBlock(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.DeleteBlock: %w", err)
	}
	return &DeleteResult{Deleted: true}, nil
}

// ListSlots lists all slots for a tenant/school/year
func (s *ServiceImpl) ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error) {
	slots, err := s.repo.ListSlots(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.ListSlots: %w", err)
	}
	return slots, nil
}

// GetSlot retrieves a single slot by ID
func (s *ServiceImpl) GetSlot(ctx context.Context, id string, tenantID, schoolID string) (*Slot, error) {
	slot, err := s.repo.GetSlot(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.GetSlot: %w", err)
	}
	return slot, nil
}

// CreateSlot creates a new slot
func (s *ServiceImpl) CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error) {
	slot, err := s.repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateSlot: %w", err)
	}
	return slot, nil
}

// BatchCreateSlots creates multiple slots at once
func (s *ServiceImpl) BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, ps []SlotPayload) ([]Slot, error) {
	slots, err := s.repo.BatchCreateSlots(ctx, tenantID, schoolID, academicYearID, ps)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.BatchCreateSlots: %w", err)
	}
	return slots, nil
}

// UpdateSlot updates an existing slot
func (s *ServiceImpl) UpdateSlot(ctx context.Context, id string, tenantID, schoolID string, p UpdateSlotPayload) (*Slot, error) {
	slot, err := s.repo.UpdateSlot(ctx, id, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.UpdateSlot: %w", err)
	}
	return slot, nil
}

// DeleteSlot deletes a slot by ID
func (s *ServiceImpl) DeleteSlot(ctx context.Context, id, tenantID, schoolID string) error {
	err := s.repo.DeleteSlot(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("timetable.ServiceImpl.DeleteSlot: %w", err)
	}
	return nil
}

// CreateTrack creates a new timetable track
func (s *ServiceImpl) CreateTrack(ctx context.Context, tenantID, schoolID, academicYearID, academicTermID, name, description string, isDefault bool) (*Track, error) {
	track, err := s.repo.CreateTrack(ctx, tenantID, schoolID, academicYearID, academicTermID, name, description, isDefault)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateTrack: %w", err)
	}
	return track, nil
}

// UpdateTrack updates an existing track
func (s *ServiceImpl) UpdateTrack(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error) {
	track, err := s.repo.UpdateTrack(ctx, id, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.UpdateTrack: %w", err)
	}
	return track, nil
}

// DeleteTrack deletes a track by ID
func (s *ServiceImpl) DeleteTrack(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	err := s.repo.DeleteTrack(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.DeleteTrack: %w", err)
	}
	return &DeleteResult{Deleted: true}, nil
}

// CreateAllocation creates a new allocation
func (s *ServiceImpl) CreateAllocation(ctx context.Context, tenantID, schoolID, blockID string, p CreateAllocationPayload) (*Allocation, error) {
	allocation, err := s.repo.CreateAllocation(ctx, tenantID, schoolID, blockID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateAllocation: %w", err)
	}
	return allocation, nil
}

// UpdateAllocation updates an existing allocation
func (s *ServiceImpl) UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error) {
	allocation, err := s.repo.UpdateAllocation(ctx, id, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.UpdateAllocation: %w", err)
	}
	return allocation, nil
}

// DeleteAllocation deletes an allocation by ID
func (s *ServiceImpl) DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error {
	err := s.repo.DeleteAllocation(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("timetable.ServiceImpl.DeleteAllocation: %w", err)
	}
	return nil
}
