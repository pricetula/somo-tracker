package timetablestructure

import (
	"context"
	"errors"
	"fmt"
)

// Service contains business logic for the timetablestructure domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// ListBlocksByDay returns all time blocks for a given day of the week.
func (s *Service) ListBlocksByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) (*TimeBlockListResult, error) {
	if tenantID == "" || schoolID == "" || academicYearID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.ListBlocksByDay: %w", ErrInvalidInput)
	}
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return nil, fmt.Errorf("timetablestructure.Service.ListBlocksByDay: %w", ErrInvalidInput)
	}

	blocks, err := s.Repo.ListByDay(ctx, tenantID, schoolID, academicYearID, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.ListBlocksByDay: %w", err)
	}

	return &TimeBlockListResult{
		Items: blocks,
		Total: len(blocks),
	}, nil
}

// ListAllBlocks returns all time blocks for a school grouped by day.
func (s *Service) ListAllBlocks(ctx context.Context, tenantID, schoolID, academicYearID string) (*TimeBlockListResult, error) {
	if tenantID == "" || schoolID == "" || academicYearID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.ListAllBlocks: %w", ErrInvalidInput)
	}

	blocks, err := s.Repo.ListAll(ctx, tenantID, schoolID, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.ListAllBlocks: %w", err)
	}

	return &TimeBlockListResult{
		Items: blocks,
		Total: len(blocks),
	}, nil
}

// CreateBlock creates a new time block after validating inputs.
func (s *Service) CreateBlock(ctx context.Context, tenantID, schoolID string, payload CreateTimeBlockPayload) (*TimeBlock, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.CreateBlock: %w", ErrInvalidInput)
	}

	if err := validateBlockPayload(payload); err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.CreateBlock: %w", err)
	}

	block, err := s.Repo.Create(ctx, tenantID, schoolID, payload)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.CreateBlock: %w", err)
	}
	return block, nil
}

// BatchCreateBlocks creates multiple time blocks atomically.
func (s *Service) BatchCreateBlocks(ctx context.Context, tenantID, schoolID string, payload BatchCreateTimeBlockPayload) (*TimeBlockListResult, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.BatchCreateBlocks: %w", ErrInvalidInput)
	}

	if len(payload.Blocks) == 0 {
		return nil, fmt.Errorf("timetablestructure.Service.BatchCreateBlocks: %w", ErrInvalidInput)
	}

	for i, block := range payload.Blocks {
		if err := validateBlockPayload(block); err != nil {
			return nil, fmt.Errorf("timetablestructure.Service.BatchCreateBlocks: block %d: %w", i, err)
		}
	}

	blocks, err := s.Repo.BatchCreate(ctx, tenantID, schoolID, payload.Blocks)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.BatchCreateBlocks: %w", err)
	}

	return &TimeBlockListResult{
		Items: blocks,
		Total: len(blocks),
	}, nil
}

// ReplicateDayBlocks replicates one day's schedule to target days.
func (s *Service) ReplicateDayBlocks(ctx context.Context, tenantID, schoolID string, payload ReplicateDayPayload) (*TimeBlockListResult, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.ReplicateDayBlocks: %w", ErrInvalidInput)
	}

	if payload.SourceDay < 1 || payload.SourceDay > 7 {
		return nil, fmt.Errorf("timetablestructure.Service.ReplicateDayBlocks: %w", ErrInvalidInput)
	}

	if len(payload.TargetDays) == 0 {
		return nil, fmt.Errorf("timetablestructure.Service.ReplicateDayBlocks: at least one target day required: %w", ErrInvalidInput)
	}

	for _, d := range payload.TargetDays {
		if d < 1 || d > 7 {
			return nil, fmt.Errorf("timetablestructure.Service.ReplicateDayBlocks: invalid target day %d: %w", d, ErrInvalidInput)
		}
	}

	blocks, err := s.Repo.ReplicateDay(ctx, tenantID, schoolID, payload.SourceDay, payload.TargetDays)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.ReplicateDayBlocks: %w", err)
	}

	return &TimeBlockListResult{
		Items: blocks,
		Total: len(blocks),
	}, nil
}

// UpdateBlock updates an existing time block.
func (s *Service) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, payload CreateTimeBlockPayload) (*TimeBlock, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", ErrInvalidInput)
	}

	if err := validateBlockPayload(payload); err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", err)
	}

	block, err := s.Repo.Update(ctx, id, tenantID, schoolID, payload)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", err)
	}
	return block, nil
}

// DeleteBlock deletes a time block by ID.
func (s *Service) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlock: %w", ErrInvalidInput)
	}

	count, err := s.Repo.HasLinkedLessons(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlock: %w", err)
	}
	if count > 0 {
		return &DeleteResult{
			Deleted:       false,
			LinkedLessons: count,
			Message:       fmt.Sprintf("Cannot delete this structural time block. It is currently linked to %d live scheduled lesson(s) on the school timetable. Please remove or clear those scheduled classes first.", count),
		}, fmt.Errorf("timetablestructure.Service.DeleteBlock: %w (linked to %d lessons)", ErrBlockHasLessons, count)
	}

	if err := s.Repo.Delete(ctx, id, tenantID, schoolID); err != nil {
		if errors.Is(err, ErrBlockHasLessons) {
			return nil, err
		}
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlock: %w", err)
	}

	return &DeleteResult{
		Deleted: true,
		Message: "Time block removed successfully",
	}, nil
}

// DeleteDayBlocks deletes all blocks for a specific day.
func (s *Service) DeleteDayBlocks(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) error {
	if tenantID == "" || schoolID == "" || academicYearID == "" {
		return fmt.Errorf("timetablestructure.Service.DeleteDayBlocks: %w", ErrInvalidInput)
	}
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return fmt.Errorf("timetablestructure.Service.DeleteDayBlocks: %w", ErrInvalidInput)
	}

	return s.Repo.DeleteByDay(ctx, tenantID, schoolID, academicYearID, dayOfWeek)
}

// validateBlockPayload validates the create/update payload fields.
func validateBlockPayload(payload CreateTimeBlockPayload) error {
	if payload.DayOfWeek < 1 || payload.DayOfWeek > 7 {
		return fmt.Errorf("%w: day_of_week must be between 1 (Monday) and 7 (Sunday)", ErrInvalidInput)
	}
	if payload.PeriodName == "" {
		return fmt.Errorf("%w: period_name is required", ErrInvalidInput)
	}
	if payload.StartTime == "" {
		return fmt.Errorf("%w: start_time is required", ErrInvalidInput)
	}
	if payload.EndTime == "" {
		return fmt.Errorf("%w: end_time is required", ErrInvalidInput)
	}
	if payload.StartTime >= payload.EndTime {
		return fmt.Errorf("%w: end_time must be after start_time", ErrInvalidInput)
	}
	if payload.AcademicYearID == "" {
		return fmt.Errorf("%w: academic_year_id is required", ErrInvalidInput)
	}
	return nil
}
