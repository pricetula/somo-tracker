// Package cbctimetableslots manages the Grid Allocation Layer — a lightweight
// relational mapping table (cbc_timetable_slots) that links classes, teachers,
// learning areas, and rooms to structural time blocks. The time range is
// inherited from the referenced timetable_structures row; this package only
// handles assignments and constraint enforcement via fast B-Tree unique indexes.
package cbctimetableslots

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound            = fmt.Errorf("cbctimetableslots not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists       = fmt.Errorf("cbctimetableslots already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput        = fmt.Errorf("invalid cbctimetableslots input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized        = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden           = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict            = fmt.Errorf("cbctimetableslots conflict: %w", middleware.ErrConflict)
	ErrTeacherDoubleBooked = fmt.Errorf("teacher already assigned during this period: %w", middleware.ErrConflict)
	ErrRoomDoubleBooked    = fmt.Errorf("room already assigned during this period: %w", middleware.ErrConflict)
	ErrClassSlotOccupied   = fmt.Errorf("class already has an assignment for this period: %w", middleware.ErrConflict)
)

// TimetableSlot represents a single allocation (class → teacher → area → room) for a structure block.
type TimetableSlot struct {
	ID             string    `json:"id"`
	AcademicYearID string    `json:"academic_year_id"`
	StructureID    string    `json:"structure_id"`
	ClassID        string    `json:"class_id"`
	LearningAreaID *string   `json:"learning_area_id,omitempty"`
	TeacherID      *string   `json:"teacher_id,omitempty"`
	RoomIdentifier *string   `json:"room_identifier,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
}

// SlotListResult holds the response for listing slots.
type SlotListResult struct {
	Items []TimetableSlot `json:"items"`
	Total int             `json:"total"`
}

// CreateSlotPayload is the request body for creating a single slot assignment.
type CreateSlotPayload struct {
	AcademicYearID string  `json:"academic_year_id"`
	StructureID    string  `json:"structure_id"`
	ClassID        string  `json:"class_id"`
	LearningAreaID *string `json:"learning_area_id,omitempty"`
	TeacherID      *string `json:"teacher_id,omitempty"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
}

// BatchCreateSlotsPayload is the request body for bulk slot operations.
type BatchCreateSlotsPayload struct {
	Slots []CreateSlotPayload `json:"slots"`
}

// UpdateSlotPayload is the request body for updating an existing slot.
type UpdateSlotPayload struct {
	LearningAreaID *string `json:"learning_area_id,omitempty"`
	TeacherID      *string `json:"teacher_id,omitempty"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
}

// SlotFilter are query parameters for filtering the slot list.
type SlotFilter struct {
	AcademicYearID string `json:"academic_year_id"`
	StructureID    string `json:"structure_id,omitempty"`
	ClassID        string `json:"class_id,omitempty"`
	TeacherID      string `json:"teacher_id,omitempty"`
	RoomIdentifier string `json:"room_identifier,omitempty"`
}

// SlotWithEnrichedData extends TimetableSlot with joined data from related tables.
type SlotWithEnrichedData struct {
	TimetableSlot
	ClassName        string  `json:"class_name,omitempty"`
	PeriodName       string  `json:"period_name,omitempty"`
	DayOfWeek        int     `json:"day_of_week,omitempty"`
	StartTime        string  `json:"start_time,omitempty"`
	EndTime          string  `json:"end_time,omitempty"`
	IsBreak          bool    `json:"is_break,omitempty"`
	LearningAreaName *string `json:"learning_area_name,omitempty"`
	TeacherName      *string `json:"teacher_name,omitempty"`
}

// EnrichedSlotListResult holds the response for enriched listing.
type EnrichedSlotListResult struct {
	Items []SlotWithEnrichedData `json:"items"`
	Total int                    `json:"total"`
}

// Repository defines the contract for timetable slot persistence.
type Repository interface {
	List(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error)
	ListEnriched(ctx context.Context, filter SlotFilter) ([]SlotWithEnrichedData, error)
	GetByID(ctx context.Context, id string) (*TimetableSlot, error)
	GetEnrichedByID(ctx context.Context, id string) (*SlotWithEnrichedData, error)
	Create(ctx context.Context, slot CreateSlotPayload) (*TimetableSlot, error)
	BatchCreate(ctx context.Context, slots []CreateSlotPayload) ([]TimetableSlot, error)
	Update(ctx context.Context, id string, slot UpdateSlotPayload) (*TimetableSlot, error)
	Delete(ctx context.Context, id string) error

	// ClearDay removes all slots for a given structure (affects all classes).
	ClearDay(ctx context.Context, structureIDs []string) error

	// ClearClassDay removes all slots for a specific class on a given structure day.
	ClearClassDay(ctx context.Context, structureID, classID string) error
}
