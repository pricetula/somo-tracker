package cbctimetable

// ─── Error types ───────────────────────────────────────────────────────────

// ConflictError indicates a scheduling conflict (teacher or room overlap).
type ConflictError struct {
	Type       string `json:"type"`       // "teacher" or "room"
	EntityName string `json:"entity"`     // teacher name or room identifier
	ClassName  string `json:"class_name"` // colliding class name
	DayOfWeek  int    `json:"day_of_week"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

func (e *ConflictError) Error() string {
	if e.Type == "teacher" {
		return e.EntityName + " is already teaching " + e.ClassName + " at this time"
	}
	return e.EntityName + " is in use by " + e.ClassName + " at this time"
}

// ─── Request / Response types ──────────────────────────────────────────────

// TimetableSlot represents a single row in cbc_timetable_slots.
type TimetableSlot struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"tenant_id"`
	SchoolID       string  `json:"school_id"`
	AcademicYearID string  `json:"academic_year_id"`
	ClassID        string  `json:"class_id"`
	TeacherID      string  `json:"teacher_id"`
	LearningAreaID *string `json:"cbc_learning_area_id"`
	RoomIdentifier *string `json:"room_identifier"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
}

// CreateSlotRequest is the request body for creating a new timetable slot.
type CreateSlotRequest struct {
	ClassID        string  `json:"class_id"`
	TeacherID      string  `json:"teacher_id"`
	LearningAreaID *string `json:"cbc_learning_area_id"`
	RoomIdentifier *string `json:"room_identifier"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
}

// UpdateSlotRequest is the request body for updating a timetable slot.
type UpdateSlotRequest struct {
	TeacherID      string  `json:"teacher_id"`
	LearningAreaID *string `json:"cbc_learning_area_id"`
	RoomIdentifier *string `json:"room_identifier"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
}

// ConflictCheckRequest is the query params for the conflict pre-check endpoint.
type ConflictCheckRequest struct {
	TeacherID      string  `json:"teacher_id"`
	DayOfWeek      int     `json:"day_of_week"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	AcademicYearID string  `json:"academic_year_id"`
	SchoolID       string  `json:"school_id"`
	RoomIdentifier *string `json:"room_identifier,omitempty"`
	ExcludeSlotID  *string `json:"exclude_slot_id,omitempty"`
	ExcludeClassID *string `json:"exclude_class_id,omitempty"`
}

// DuplicateDayRequest is the request body for duplicating a day's slots.
type DuplicateDayRequest struct {
	SourceDay      int    `json:"source_day"`
	TargetDays     []int  `json:"target_days"`
	AcademicYearID string `json:"academic_year_id"`
	ClassID        string `json:"class_id"`
}

// CopyFromClassRequest is the request body for copying timetable from another class.
type CopyFromClassRequest struct {
	SourceClassID  string `json:"source_class_id"`
	AcademicYearID string `json:"academic_year_id"`
	TargetClassID  string `json:"target_class_id"`
}

// BulkOperationResult describes the outcome of a bulk copy/duplicate operation.
type BulkOperationResult struct {
	TotalCopied int              `json:"total_copied"`
	Skipped     []SlotSkipReason `json:"skipped"`
}

// SlotSkipReason describes why an individual slot was skipped during a bulk op.
type SlotSkipReason struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	Reason    string `json:"reason"`
}

// AttendanceCount holds the count of attendance periods linked to a slot.
type AttendanceCount struct {
	Count int `json:"count"`
}

// ─── DB-level conflict info (raw from queries) ─────────────────────────────

// conflictingSlot is returned by the overlap queries.
type conflictingSlot struct {
	ID          string
	ClassName   string
	StartTime   string
	EndTime     string
	TeacherID   string
	TeacherName string
	RoomID      *string
}

// ClassBrief is a lightweight class reference.
type ClassBrief struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GradeName string `json:"grade_name"`
	Stream    string `json:"stream"`
}

// LearningAreaBrief is a lightweight learning area reference.
type LearningAreaBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// TeacherBrief is a lightweight teacher reference.
type TeacherBrief struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// SlotBrief is a lightweight slot with resolved names for the attendance page.
type SlotBrief struct {
	PeriodID         string `json:"period_id"`
	LearningAreaName string `json:"learning_area_name"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
}
