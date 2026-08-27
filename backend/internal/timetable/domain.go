package timetable

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

var (
	ErrNotFound                = xerrors.NotFound("timetable not found")
	ErrAlreadyExists           = xerrors.AlreadyExists("timetable already exists")
	ErrInvalidInput            = xerrors.InvalidInput("invalid timetable input")
	ErrUnauthorized            = xerrors.Unauthorized("unauthorized")
	ErrForbidden               = xerrors.Forbidden("forbidden")
	ErrConflict                = xerrors.Conflict("timetable conflict")
	ErrTeacherDoubleBooked     = xerrors.Conflict("teacher already assigned during this period")
	ErrRoomDoubleBooked        = xerrors.Conflict("room already assigned during this period")
	ErrClassAllocationOccupied = xerrors.Conflict("class already has an allocation for this period")
	ErrBlockOverlap            = xerrors.Conflict("time block collides with an existing block")
	ErrBlockHasLessons         = xerrors.Conflict("time block is linked to live scheduled lessons")
)

type TimeBlock struct {
	ID             string    `json:"id"`
	TrackID        string    `json:"track_id"`
	DayOfWeek      int       `json:"day_of_week"`
	PeriodName     string    `json:"period_name"`
	StartTime      string    `json:"start_time"`
	EndTime        string    `json:"end_time"`
	IsBreak        bool      `json:"is_break"`
	OrderIndex     int       `json:"order_index"`
	AcademicYearID string    `json:"academic_year_id,omitempty"` // derived from track
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// Allocation is the full allocation record with joined names (returned by GET endpoints)
type Allocation struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SchoolID       string    `json:"school_id"`
	AcademicYearID string    `json:"academic_year_id"` // derived from block -> track
	BlockID        string    `json:"block_id"`
	ClassID        string    `json:"class_id"`
	LearningAreaID string    `json:"learning_area_id"`
	TeacherID      string    `json:"teacher_id"`
	RoomIdentifier *string   `json:"room_identifier,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`

	// Joined fields (always populated by ListAllocations)
	ClassName        string `json:"class_name"`
	LearningAreaName string `json:"learning_area_name"`
	LearningAreaCode string `json:"learning_area_code"`
	TeacherName      string `json:"teacher_name"`
	RoomName         string `json:"room_name,omitempty"`
}

type AllocationFilter struct {
	TenantID       string `json:"tenant_id"`
	SchoolID       string `json:"school_id"`
	AcademicYearID string `json:"academic_year_id"`
	TrackID        string `json:"track_id,omitempty"`
	BlockID        string `json:"block_id,omitempty"`
	ClassID        string `json:"class_id,omitempty"`
	TeacherID      string `json:"teacher_id,omitempty"`
	LearningAreaID string `json:"learning_area_id,omitempty"`
}

type TimeBlockListResult struct {
	Items []TimeBlock `json:"items"`
	Total int         `json:"total"`
}

type AllocationListResult struct {
	Items []Allocation `json:"items"`
	Total int          `json:"total"`
}

// Payload types (input only — no joined fields)
type CreateTimeBlockPayload struct {
	TrackID    string `json:"track_id" validate:"required"`
	DayOfWeek  int    `json:"day_of_week"`
	PeriodName string `json:"period_name"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	IsBreak    bool   `json:"is_break"`
	OrderIndex int    `json:"order_index"`
}

type UpdateTimeBlockPayload struct {
	ID         string `json:"id" validate:"required"`
	TrackID    string `json:"track_id" validate:"required"`
	DayOfWeek  int    `json:"day_of_week"`
	PeriodName string `json:"period_name"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	IsBreak    bool   `json:"is_break"`
	OrderIndex int    `json:"order_index"`
}

// UpdatePeriodPayload updates ALL blocks (across 7 days) that share the same
// (track_id, period_name) tuple. Used to edit a logical period once and have
// the change apply to every day-of-week replication.
type UpdatePeriodPayload struct {
	TrackID    string `json:"track_id" validate:"required"`
	PeriodName string `json:"period_name" validate:"required"`
	StartTime  string `json:"start_time,omitempty"`
	EndTime    string `json:"end_time,omitempty"`
	IsBreak    *bool  `json:"is_break,omitempty"`
}

// DeletePeriodPayload deletes ALL blocks across 7 days for the given
// (track_id, period_name). Cascades to any allocations referencing the blocks.
type DeletePeriodPayload struct {
	TrackID     string   `json:"track_id" validate:"required"`
	PeriodName  string   `json:"period_name" validate:"required"`
	PeriodNames []string `json:"period_names,omitempty"` // optional bulk variant
}

type CreateAllocationPayload struct {
	BlockID        string  `json:"block_id" validate:"required"`
	ClassID        string  `json:"class_id" validate:"required"`
	LearningAreaID string  `json:"learning_area_id" validate:"required"`
	TeacherID      string  `json:"teacher_id" validate:"required"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
}

type UpdateAllocationPayload struct {
	ID             string  `json:"id" validate:"required"`
	ClassID        string  `json:"class_id,omitempty"`
	LearningAreaID string  `json:"learning_area_id,omitempty"`
	TeacherID      string  `json:"teacher_id,omitempty"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
}

type DeleteResult struct {
	Deleted       bool   `json:"deleted"`
	DeletedCount  int    `json:"deleted_count,omitempty"`
	LinkedLessons int    `json:"linked_lessons,omitempty"`
	Message       string `json:"message,omitempty"`
}

type Repository interface {
	ListBlocks(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error)
	GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error)
	CreateBlock(ctx context.Context, tenantID, schoolID string, payload CreateTimeBlockPayload) (*TimeBlock, error)
	UpdateBlock(ctx context.Context, id, tenantID, schoolID string, payload UpdateTimeBlockPayload) (*TimeBlock, error)
	DeleteBlock(ctx context.Context, id, tenantID, schoolID string) error
	GetTrack(ctx context.Context, id, tenantID, schoolID string) (*Track, error)
	UpdateBlockPeriod(ctx context.Context, tenantID, schoolID string, p UpdatePeriodPayload) ([]TimeBlock, error)
	DeleteBlockPeriod(ctx context.Context, tenantID, schoolID string, p DeletePeriodPayload) (*DeleteResult, error)

	ListAllocations(ctx context.Context, filter AllocationFilter) ([]Allocation, error)
	GetAllocation(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error)
	CreateAllocation(ctx context.Context, tenantID, schoolID string, payload CreateAllocationPayload) (*Allocation, error)
	BatchCreateAllocations(ctx context.Context, tenantID, schoolID string, payloads []CreateAllocationPayload) ([]Allocation, error)
	UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error)
	DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error

	// Track
	ListTracks(ctx context.Context, tenantID, schoolID, academicYearID string) ([]Track, error)
	CreateTrack(ctx context.Context, tenantID, schoolID, academicYearID, academicTermID, name, description string, isDefault bool) (*Track, error)
	UpdateTrack(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error)
	DeleteTrack(ctx context.Context, id, tenantID, schoolID string) error
}

type Service interface {
	ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error)
	GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error)
	CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error)
	UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error)
	DeleteBlock(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error)
	UpdateBlockPeriod(ctx context.Context, tenantID, schoolID string, p UpdatePeriodPayload) ([]TimeBlock, error)
	DeleteBlockPeriod(ctx context.Context, tenantID, schoolID string, p DeletePeriodPayload) (*DeleteResult, error)

	ListAllocations(ctx context.Context, f AllocationFilter) ([]Allocation, error)
	GetAllocation(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error)
	CreateAllocation(ctx context.Context, tenantID, schoolID string, p CreateAllocationPayload) (*Allocation, error)
	BatchCreateAllocations(ctx context.Context, tenantID, schoolID string, ps []CreateAllocationPayload) ([]Allocation, error)
	UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error)
	DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error

	// Track
	ListTracks(ctx context.Context, tenantID, schoolID, academicYearID string) ([]Track, error)
	GetTrack(ctx context.Context, id, tenantID, schoolID string) (*Track, error)
	CreateTrack(ctx context.Context, tenantID, schoolID, academicYearID, academicTermID, name, description string, isDefault bool) (*Track, error)
	UpdateTrack(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error)
	DeleteTrack(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error)
}

type Track struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SchoolID       string    `json:"school_id"`
	AcademicYearID string    `json:"academic_year_id"`
	AcademicTermID string    `json:"academic_term_id,omitempty"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	IsDefault      bool      `json:"is_default"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type CreateTrackPayload struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	IsDefault     bool                     `json:"is_default,omitempty"`
	InitialBlocks []CreateTimeBlockPayload `json:"initial_blocks,omitempty"`
}

type UpdateTrackPayload struct {
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IsDefault   *bool  `json:"is_default,omitempty"`
}
