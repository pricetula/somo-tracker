// Package timetablestructure manages the structural day template — the Grid
// Definition Layer. Holds time ranges and rules (period_name, is_break) that
// define a standard school day per academic year. Decoupled from individual
// allocation slots (cbc_timetable_slots) which reference structure_id instead
// of carrying raw time ranges.
package timetablestructure

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound        = fmt.Errorf("timetablestructure not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists   = fmt.Errorf("timetablestructure already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput    = fmt.Errorf("invalid timetablestructure input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized    = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden       = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict        = fmt.Errorf("timetablestructure conflict: %w", middleware.ErrConflict)
	ErrBlockOverlap    = fmt.Errorf("time block collides with an existing block: %w", middleware.ErrConflict)
	ErrBlockHasLessons = fmt.Errorf("time block is linked to live scheduled lessons: %w", middleware.ErrConflict)
)

// DayOfWeek constants map directly to PostgreSQL weekday numbers.
const (
	DayMonday    = 1
	DayTuesday   = 2
	DayWednesday = 3
	DayThursday  = 4
	DayFriday    = 5
	DaySaturday  = 6
	DaySunday    = 7
)

// TimeBlock represents a single structural time block in a school day.
type TimeBlock struct {
	ID             string    `json:"id"`
	DayOfWeek      int       `json:"day_of_week"`
	PeriodName     string    `json:"period_name"`
	StartTime      string    `json:"start_time"`
	EndTime        string    `json:"end_time"`
	IsBreak        bool      `json:"is_break"`
	AcademicYearID string    `json:"academic_year_id,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// TimeBlockListResult holds the response for listing time blocks.
type TimeBlockListResult struct {
	Items []TimeBlock `json:"items"`
	Total int         `json:"total"`
}

// CreateTimeBlockPayload is the request body for creating a time block.
type CreateTimeBlockPayload struct {
	DayOfWeek      int    `json:"day_of_week"`
	PeriodName     string `json:"period_name"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	IsBreak        bool   `json:"is_break"`
	AcademicYearID string `json:"academic_year_id"`
}

// BatchCreateTimeBlockPayload is the request body for batch-importing a
// standard day template (e.g. "Standard Friday Matrix") that creates
// multiple time blocks atomically.
type BatchCreateTimeBlockPayload struct {
	Blocks []CreateTimeBlockPayload `json:"blocks"`
}

// ReplicateDayPayload is the request body for replicating one day's schedule
// to other weekdays (the "Mass Replication" ROI feature).
type ReplicateDayPayload struct {
	SourceDay  int   `json:"source_day"`
	TargetDays []int `json:"target_days"`
}

// DeleteResult carries the result of a delete attempt.
type DeleteResult struct {
	Deleted       bool   `json:"deleted"`
	LinkedLessons int    `json:"linked_lessons,omitempty"`
	Message       string `json:"message,omitempty"`
}

// Repository defines the contract for timetable structure persistence.
type Repository interface {
	ListByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) ([]TimeBlock, error)
	ListAll(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error)
	Create(ctx context.Context, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error)
	BatchCreate(ctx context.Context, tenantID, schoolID string, blocks []CreateTimeBlockPayload) ([]TimeBlock, error)
	ReplicateDay(ctx context.Context, tenantID, schoolID string, sourceDay int, targetDays []int) ([]TimeBlock, error)
	Update(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error)
	Delete(ctx context.Context, id, tenantID, schoolID string) error
	DeleteByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) error

	// HasLinkedLessons checks whether any cbc_timetable_slots reference this
	// structural time block.
	HasLinkedLessons(ctx context.Context, id, tenantID, schoolID string) (int, error)

	// FindOverlappingBlock returns the first block that overlaps with the
	// given time range on the same day, excluding the block with the given ID
	// (used during updates). Returns nil if no overlap exists.
	FindOverlappingBlock(ctx context.Context, tenantID, schoolID string, dayOfWeek int, startTime, endTime string, excludeID string) (*TimeBlock, error)
}
