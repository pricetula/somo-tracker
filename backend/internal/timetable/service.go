package timetable

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"somotracker/backend/internal/xerrors"
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

// ListAllocations lists all allocations with joined names for a tenant/school/year
func (s *ServiceImpl) ListAllocations(ctx context.Context, f AllocationFilter) ([]Allocation, error) {
	allocations, err := s.repo.ListAllocations(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.ListAllocations: %w", err)
	}
	return allocations, nil
}

// GetAllocation retrieves a single allocation by ID with joined names
func (s *ServiceImpl) GetAllocation(ctx context.Context, id string, tenantID, schoolID string) (*Allocation, error) {
	allocation, err := s.repo.GetAllocation(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.GetAllocation: %w", err)
	}
	return allocation, nil
}

// CreateAllocation creates a new allocation
func (s *ServiceImpl) CreateAllocation(ctx context.Context, tenantID, schoolID string, p CreateAllocationPayload) (*Allocation, error) {
	// Reject allocations on break blocks
	block, err := s.repo.GetBlock(ctx, p.BlockID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateAllocation: get block: %w", err)
	}
	if block.IsBreak {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateAllocation: %w", xerrors.Conflict("cannot assign to a break block"))
	}

	allocation, err := s.repo.CreateAllocation(ctx, tenantID, schoolID, p)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.CreateAllocation: %w", err)
	}
	return allocation, nil
}

// BatchCreateAllocations creates multiple allocations at once
func (s *ServiceImpl) BatchCreateAllocations(ctx context.Context, tenantID, schoolID string, ps []CreateAllocationPayload) ([]Allocation, error) {
	allocations, err := s.repo.BatchCreateAllocations(ctx, tenantID, schoolID, ps)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.BatchCreateAllocations: %w", err)
	}
	return allocations, nil
}

// UpdateAllocation updates an existing allocation
func (s *ServiceImpl) UpdateAllocation(ctx context.Context, id string, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error) {
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

// ListTracks lists all timetable tracks for a school/year
func (s *ServiceImpl) ListTracks(ctx context.Context, tenantID, schoolID, yearID string) ([]Track, error) {
	tracks, err := s.repo.ListTracks(ctx, tenantID, schoolID, yearID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.ListTracks: %w", err)
	}
	return tracks, nil
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

func (s *ServiceImpl) UpdateBlockPeriod(ctx context.Context, tenantID, schoolID string, p UpdatePeriodPayload) ([]TimeBlock, error) {
	return s.repo.UpdateBlockPeriod(ctx, tenantID, schoolID, p)
}

func (s *ServiceImpl) DeleteBlockPeriod(ctx context.Context, tenantID, schoolID string, p DeletePeriodPayload) (*DeleteResult, error) {
	return s.repo.DeleteBlockPeriod(ctx, tenantID, schoolID, p)
}

// DeleteTrack deletes a track by ID
func (s *ServiceImpl) DeleteTrack(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	err := s.repo.DeleteTrack(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetable.ServiceImpl.DeleteTrack: %w", err)
	}
	return &DeleteResult{Deleted: true}, nil
}
