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

// UpdateBlock updates a time block with optional cascade (same period name
// on all days) and shift-following (adjust subsequent blocks on the same day).
func (s *Service) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, payload UpdateTimeBlockPayload) (*TimeBlockListResult, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", ErrInvalidInput)
	}

	// Validate core fields
	if payload.DayOfWeek < 1 || payload.DayOfWeek > 7 {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: day_of_week out of range: %w", ErrInvalidInput)
	}
	if payload.PeriodName == "" {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", ErrInvalidInput)
	}
	if payload.StartTime == "" || payload.EndTime == "" {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", ErrInvalidInput)
	}
	if payload.StartTime >= payload.EndTime {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: end_time must be after start_time: %w", ErrInvalidInput)
	}
	if payload.AcademicYearID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", ErrInvalidInput)
	}

	// Validate propagate mode
	if payload.Propagate != "" && payload.Propagate != "all_days" {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: invalid propagate mode %q: %w", payload.Propagate, ErrInvalidInput)
	}

	// ── Step 1: Extract the base update payload ──
	basePayload := CreateTimeBlockPayload{
		DayOfWeek:      payload.DayOfWeek,
		PeriodName:     payload.PeriodName,
		StartTime:      payload.StartTime,
		EndTime:        payload.EndTime,
		IsBreak:        payload.IsBreak,
		AcademicYearID: payload.AcademicYearID,
	}

	// ── Step 2: Get the current block (for delta calculation) ──
	currentBlock, err := s.Repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: get current: %w", err)
	}

	// ── Step 3: Perform the primary update ──
	updated, err := s.Repo.Update(ctx, id, tenantID, schoolID, basePayload)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: %w", err)
	}

	allBlocks := []TimeBlock{*updated}

	// ── Step 4: Cascade — apply same change to same-named blocks on other days ──
	if payload.Propagate == "all_days" {
		sameName, err := s.Repo.ListByPeriodName(ctx, tenantID, schoolID, payload.AcademicYearID, payload.PeriodName, id)
		if err != nil {
			return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: list by name: %w", err)
		}

		if len(sameName) > 0 {
			var batch []BatchBlockUpdate
			for _, b := range sameName {
				batch = append(batch, BatchBlockUpdate{
					ID:        b.ID,
					StartTime: payload.StartTime,
					EndTime:   payload.EndTime,
				})
			}

			cascaded, err := s.Repo.BatchUpdateBlocks(ctx, tenantID, schoolID, batch)
			if err != nil {
				return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: cascade: %w", err)
			}
			allBlocks = append(allBlocks, cascaded...)
		}
	}

	// ── Step 5: Shift following — slide subsequent blocks on the same day ──
	if payload.ShiftFollowing {
		// Calculate delta in minutes: new_start - old_start
		oldStart := parseTimeMinutes(currentBlock.StartTime)
		newStart := parseTimeMinutes(payload.StartTime)
		delta := newStart - oldStart

		if delta != 0 {
			// Find blocks after the old end_time on the same day, excluding the current block
			subsequent, err := s.Repo.ListByDayAfter(ctx, tenantID, schoolID, payload.AcademicYearID, currentBlock.DayOfWeek, currentBlock.EndTime, id)
			if err != nil {
				return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: list subsequent: %w", err)
			}

			if len(subsequent) > 0 {
				var batchShift []BatchBlockUpdate
				for _, b := range subsequent {
					bStart := parseTimeMinutes(b.StartTime) + delta
					bEnd := parseTimeMinutes(b.EndTime) + delta

					// Clamp to valid day (24h max)
					if bStart < 0 {
						bStart = 0
					}
					if bEnd > 24*60 {
						bEnd = 24 * 60
					}

					batchShift = append(batchShift, BatchBlockUpdate{
						ID:        b.ID,
						StartTime: formatMinutes(bStart),
						EndTime:   formatMinutes(bEnd),
					})
				}

				shifted, err := s.Repo.BatchUpdateBlocks(ctx, tenantID, schoolID, batchShift)
				if err != nil {
					return nil, fmt.Errorf("timetablestructure.Service.UpdateBlock: shift: %w", err)
				}
				allBlocks = append(allBlocks, shifted...)
			}
		}
	}

	return &TimeBlockListResult{Items: allBlocks, Total: len(allBlocks)}, nil
}

// parseTimeMinutes converts "HH:MM" to minutes since midnight.
func parseTimeMinutes(t string) int {
	if len(t) < 5 {
		return 0
	}
	return (int(t[0]-'0')*10+int(t[1]-'0'))*60 + int(t[3]-'0')*10 + int(t[4]-'0')
}

// formatMinutes converts minutes since midnight to "HH:MM".
func formatMinutes(m int) string {
	if m < 0 {
		m = 0
	}
	if m >= 24*60 {
		m = 24*60 - 1
	}
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
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

// DeleteBlocksByName deletes all time blocks with the given period name.
// Returns an error if any of the blocks have linked lessons.
func (s *Service) DeleteBlocksByName(ctx context.Context, tenantID, schoolID string, payload DeleteByNamePayload) (*DeleteResult, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlocksByName: %w", ErrInvalidInput)
	}
	if payload.PeriodName == "" || payload.AcademicYearID == "" {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlocksByName: %w", ErrInvalidInput)
	}

	// First, find all matching blocks to get their IDs for lesson checking
	blocks, err := s.Repo.ListByPeriodName(ctx, tenantID, schoolID, payload.AcademicYearID, payload.PeriodName, "")
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlocksByName: list: %w", err)
	}

	if len(blocks) == 0 {
		return &DeleteResult{
			Deleted: false,
			Message: fmt.Sprintf("No time blocks found with period name '%s'", payload.PeriodName),
		}, nil
	}

	// Check linked lessons for all matching blocks
	ids := make([]string, len(blocks))
	for i, b := range blocks {
		ids[i] = b.ID
	}

	totalLinked, err := s.Repo.HasLinkedLessonsForBlocks(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlocksByName: check linked: %w", err)
	}
	if totalLinked > 0 {
		return &DeleteResult{
			Deleted:       false,
			DeletedCount:  0,
			LinkedLessons: totalLinked,
			Message:       fmt.Sprintf("Cannot delete blocks named '%s' — %d of them are linked to scheduled lessons. Remove those assignments first.", payload.PeriodName, totalLinked),
		}, fmt.Errorf("timetablestructure.Service.DeleteBlocksByName: %w (%d linked lessons)", ErrBlockHasLessons, totalLinked)
	}

	deleted, err := s.Repo.DeleteByPeriodName(ctx, tenantID, schoolID, payload.AcademicYearID, payload.PeriodName)
	if err != nil {
		return nil, fmt.Errorf("timetablestructure.Service.DeleteBlocksByName: %w", err)
	}

	return &DeleteResult{
		Deleted:      true,
		DeletedCount: deleted,
		Message:      fmt.Sprintf("'%s' blocks removed from all days (%d deleted)", payload.PeriodName, deleted),
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
