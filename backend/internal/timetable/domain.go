package timetable

import (
	"context"
	"errors"
	"time"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("timetable not found")
	ErrAlreadyExists = errors.New("timetable already exists")
	ErrInvalidInput  = errors.New("invalid timetable input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("timetable conflict")
)

// TeacherRole mirrors the PostgreSQL teacher_role enum.
type TeacherRole string

const (
	TeacherRolePrimary    TeacherRole = "PRIMARY_CLASS_TEACHER"
	TeacherRoleSubject    TeacherRole = "SUBJECT_TEACHER"
	TeacherRoleSubstitute TeacherRole = "SUBSTITUTE_TEACHER"
)

// TimetableSlot represents a row in cbc_timetable_slots.
type TimetableSlot struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"tenant_id"`
	SchoolID       string  `json:"school_id"`
	AcademicYearID string  `json:"academic_year_id"`
	AcademicTermID string  `json:"academic_term_id"`
	ClassID        string  `json:"class_id"`
	TeacherID      string  `json:"teacher_id"`
	LearningAreaID *string `json:"learning_area_id,omitempty"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
}

// CreateTimetableSlotInput is the payload for creating or updating a single slot.
type CreateTimetableSlotInput struct {
	ClassID        string  `json:"class_id"`
	TeacherID      string  `json:"teacher_id"`
	LearningAreaID *string `json:"learning_area_id,omitempty"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
}

// BulkCreateTimetableSlotsInput wraps a bulk upsert request.
type BulkCreateTimetableSlotsInput struct {
	AcademicYearID string                     `json:"academic_year_id"`
	AcademicTermID string                     `json:"academic_term_id"`
	Slots          []CreateTimetableSlotInput `json:"slots"`
}

// ClassTeacher represents a row in cbc_class_teachers.
type ClassTeacher struct {
	ID             string      `json:"id"`
	TenantID       string      `json:"tenant_id"`
	ClassID        string      `json:"class_id"`
	UserID         string      `json:"user_id"`
	LearningAreaID *string     `json:"learning_area_id,omitempty"`
	TeacherRole    TeacherRole `json:"teacher_role"`
	CreatedAt      time.Time   `json:"created_at"`
}

// ClassTeacherInput is the payload for assigning a teacher to a class.
type ClassTeacherInput struct {
	TenantID       string      `json:"-"`
	SchoolID       string      `json:"-"`
	ClassID        string      `json:"class_id"`
	UserID         string      `json:"user_id"`
	LearningAreaID *string     `json:"learning_area_id,omitempty"`
	TeacherRole    TeacherRole `json:"teacher_role"`
}

// Repository defines the contract for timetable persistence.
type Repository interface {
	BulkUpsertSlots(ctx context.Context, tenantID, schoolID string, input BulkCreateTimetableSlotsInput) error
	GetSlotsByClass(ctx context.Context, tenantID, classID, termID string) ([]TimetableSlot, error)
	GetSlotsByTeacher(ctx context.Context, tenantID, teacherID, termID string) ([]TimetableSlot, error)
	AssignClassTeacher(ctx context.Context, input ClassTeacherInput) error
	RemoveClassTeacher(ctx context.Context, tenantID, classID, userID string) error
	HasPrimaryRole(ctx context.Context, tenantID, userID string) (bool, error)
	ValidateTerm(ctx context.Context, tenantID, schoolID, termID string) (bool, error)
}
