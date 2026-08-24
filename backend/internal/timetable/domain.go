package timetable

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

var (
	ErrNotFound            = xerrors.NotFound("timetable not found")
	ErrAlreadyExists       = xerrors.AlreadyExists("timetable already exists")
	ErrInvalidInput        = xerrors.InvalidInput("invalid timetable input")
	ErrUnauthorized        = xerrors.Unauthorized("unauthorized")
	ErrForbidden           = xerrors.Forbidden("forbidden")
	ErrConflict            = xerrors.Conflict("timetable conflict")
	ErrTeacherDoubleBooked = xerrors.Conflict("teacher already assigned during this period")
	ErrRoomDoubleBooked    = xerrors.Conflict("room already assigned during this period")
	ErrClassSlotOccupied   = xerrors.Conflict("class already has an assignment for this period")
	ErrBlockOverlap        = xerrors.Conflict("time block collides with an existing block")
	ErrBlockHasLessons     = xerrors.Conflict("time block is linked to live scheduled lessons")
)

type TimeBlock struct {
	ID             string    `json:"id"`
	DayOfWeek      int       `json:"day_of_week"`
	PeriodName     string    `json:"period_name"`
	StartTime      string    `json:"start_time"`
	EndTime        string    `json:"end_time"`
	IsBreak        bool      `json:"is_break"`
	AcademicYearID string    `json:"academic_year_id,omitempty"`
	Order          int       `json:"order"` // NEW — for UI sorting
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type Slot struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SchoolID       string    `json:"school_id"`
	AcademicYearID string    `json:"academic_year_id"`
	BlockID        string    `json:"block_id"`
	ClassID        string    `json:"class_id"`
	LearningAreaID string    `json:"learning_area_id"`
	TeacherID      string    `json:"teacher_id"`
	RoomIdentifier *string   `json:"room_identifier,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type SlotFilter struct {
	TenantID       string `json:"tenant_id"`
	SchoolID       string `json:"school_id"`
	AcademicYearID string `json:"academic_year_id"`
	BlockID        string `json:"block_id,omitempty"`
	ClassID        string `json:"class_id,omitempty"`
	TeacherID      string `json:"teacher_id,omitempty"`
	LearningAreaID string `json:"learning_area_id,omitempty"`
}

type TimeBlockListResult struct {
	Items []TimeBlock `json:"items"`
	Total int         `json:"total"`
}

type SlotListResult struct {
	Items []Slot `json:"items"`
	Total int    `json:"total"`
}

type CreateTimeBlockPayload struct {
	DayOfWeek      int    `json:"day_of_week"`
	PeriodName     string `json:"period_name"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	IsBreak        bool   `json:"is_break"`
	AcademicYearID string `json:"-"`
	Order          int    `json:"order"`
}

type UpdateTimeBlockPayload struct {
	DayOfWeek      int    `json:"day_of_week"`
	PeriodName     string `json:"period_name"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	IsBreak        bool   `json:"is_break"`
	AcademicYearID string `json:"academic_year_id"`
	Order          int    `json:"order"`
}

type SlotPayload struct {
	BlockID        string  `json:"block_id"`
	ClassID        string  `json:"class_id"`
	LearningAreaID string  `json:"learning_area_id"`
	TeacherID      string  `json:"teacher_id"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
}

type UpdateSlotPayload struct {
	LearningAreaID string  `json:"learning_area_id"`
	TeacherID      string  `json:"teacher_id"`
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

	ListSlots(ctx context.Context, filter SlotFilter) ([]Slot, error)
	GetSlot(ctx context.Context, id, tenantID, schoolID string) (*Slot, error)
	CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, payload SlotPayload) (*Slot, error)
	BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, payloads []SlotPayload) ([]Slot, error)
	UpdateSlot(ctx context.Context, id, tenantID, schoolID string, p UpdateSlotPayload) (*Slot, error)
	DeleteSlot(ctx context.Context, id, tenantID, schoolID string) error
}

type Service interface {
	ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error)
	GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error)
	CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error)
	UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error)
	DeleteBlock(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error)

	ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error)
	GetSlot(ctx context.Context, id, tenantID, schoolID string) (*Slot, error)
	CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error)
	BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, ps []SlotPayload) ([]Slot, error)
	UpdateSlot(ctx context.Context, id, tenantID, schoolID string, p UpdateSlotPayload) (*Slot, error)
	DeleteSlot(ctx context.Context, id, tenantID, schoolID string) error
}
