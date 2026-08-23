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

func (s *ServiceImpl) ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
	blocks, err := s.repo.ListBlocks(ctx, tenantID, schoolID, yearID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.ListBlocks: %w", err)
	}
	return blocks, nil
}

func (s *ServiceImpl) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	b, err := s.repo.GetBlock(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.GetBlock: %w", err)
	}
	return b, nil
}

func (s *ServiceImpl) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	b, err := s.repo.CreateBlock(ctx, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateBlock: %w", err)
	}
	return b, nil
}

func (s *ServiceImpl) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	b, err := s.repo.UpdateBlock(ctx, id, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.UpdateBlock: %w", err)
	}
	return b, nil
}

func (s *ServiceImpl) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	err := s.repo.DeleteBlock(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.DeleteBlock: %w", err)
	}
	return &DeleteResult{Deleted: true}, nil
}

func (s *ServiceImpl) ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error) {
	slots, err := s.repo.ListSlots(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.ListSlots: %w", err)
	}
	return slots, nil
}

func (s *ServiceImpl) GetSlot(ctx context.Context, id string) (*Slot, error) {
	slot, err := s.repo.GetSlot(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.GetSlot: %w", err)
	}
	return slot, nil
}

func (s *ServiceImpl) CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error) {
	slot, err := s.repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateSlot: %w", err)
	}
	return slot, nil
}

func (s *ServiceImpl) BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, ps []SlotPayload) ([]Slot, error) {
	slots, err := s.repo.BatchCreateSlots(ctx, tenantID, schoolID, academicYearID, ps)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.BatchCreateSlots: %w", err)
	}
	return slots, nil
}

func (s *ServiceImpl) UpdateSlot(ctx context.Context, id string, p UpdateSlotPayload) (*Slot, error) {
	slot, err := s.repo.UpdateSlot(ctx, id, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.UpdateSlot: %w", err)
	}
	return slot, nil
}

func (s *ServiceImpl) DeleteSlot(ctx context.Context, id string) error {
	err := s.repo.DeleteSlot(ctx, id)
	if err != nil {
		return fmt.Errorf("timetable.ServiceImpl.DeleteSlot: %w", err)
	}
	return nil
}
