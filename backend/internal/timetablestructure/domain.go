// Package timetablestructure manages the structural day template — the Grid
// Definition Layer. Holds time ranges and rules (period_name, is_break) that
// define a standard school day per academic year. Decoupled from individual
// allocation slots (cbc_timetable_slots) which reference structure_id instead
// of carrying raw time ranges.
package timetablestructure

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors.
var (
	ErrNotFound        = xerrors.NotFound("timetablestructure not found")
	ErrAlreadyExists   = xerrors.AlreadyExists("timetablestructure already exists")
	ErrInvalidInput    = xerrors.InvalidInput("invalid timetablestructure input")
	ErrUnauthorized    = xerrors.Unauthorized("unauthorized")
	ErrForbidden       = xerrors.Forbidden("forbidden")
	ErrConflict        = xerrors.Conflict("timetablestructure conflict")
	ErrBlockOverlap    = xerrors.Conflict("time block collides with an existing block")
	ErrBlockHasLessons = xerrors.Conflict("time block is linked to live scheduled lessons")
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
// academic_year_id is resolved server-side from the current active academic year.
type CreateTimeBlockPayload struct {
	DayOfWeek      int    `json:"day_of_week"`
	PeriodName     string `json:"period_name"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	IsBreak        bool   `json:"is_break"`
	AcademicYearID string `json:"-"` // resolved server-side, not from request body
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

// UpdateTimeBlockPayload is the request body for updating a time block,
// with optional propagation (cascade to same-named blocks on other days)
// and shift-following (adjust subsequent blocks on the same day).
type UpdateTimeBlockPayload struct {
	DayOfWeek      int    `json:"day_of_week"`
	PeriodName     string `json:"period_name"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	IsBreak        bool   `json:"is_break"`
	AcademicYearID string `json:"academic_year_id"`

	// Propagate controls which blocks are updated.
	//   ""         — update only this specific block (default)
	//   "all_days" — update all blocks with the same period_name on all days
	Propagate string `json:"propagate,omitempty"`

	// ShiftFollowing, when true, shifts all subsequent blocks on the same
	// day by the same delta as this block's time change. Requires the
	// block's original start/end times to compute the delta.
	ShiftFollowing bool `json:"shift_following,omitempty"`
}

// DeleteByNamePayload is the request body for deleting blocks by period name.
// academic_year_id is resolved server-side from the current active academic year.
type DeleteByNamePayload struct {
	PeriodName     string `json:"period_name"`
	AcademicYearID string `json:"-"` // resolved server-side, not from request body
}

// BatchBlockUpdate carries the fields needed for a single block update
// within a batch update operation.
type BatchBlockUpdate struct {
	ID        string
	StartTime string
	EndTime   string
}

// DeleteResult carries the result of a delete attempt.
type DeleteResult struct {
	Deleted       bool   `json:"deleted"`
	DeletedCount  int    `json:"deleted_count,omitempty"`
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

	// ListByPeriodName returns all time blocks with the given period name
	// within the same academic year, optionally excluding a specific block ID.
	ListByPeriodName(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error)

	// ListByDayAfter returns all time blocks on a given day whose start_time
	// is >= afterTime, optionally excluding a specific block ID.
	ListByDayAfter(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int, afterTime string, excludeID string) ([]TimeBlock, error)

	// BatchUpdateBlocks updates the start_time and end_time for a set of
	// blocks atomically within a single transaction.
	BatchUpdateBlocks(ctx context.Context, tenantID, schoolID string, blocks []BatchBlockUpdate) ([]TimeBlock, error)

	// DeleteByPeriodName removes all blocks with the given period name.
	// Returns the number of rows deleted.
	DeleteByPeriodName(ctx context.Context, tenantID, schoolID, academicYearID, periodName string) (int, error)

	// HasLinkedLessonsForBlocks checks whether any cbc_timetable_slots
	// reference any of the given structure IDs. Returns a count.
	HasLinkedLessonsForBlocks(ctx context.Context, ids []string) (int, error)
}
