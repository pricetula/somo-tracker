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

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — request / response types
// ═══════════════════════════════════════════════════════════════════════════

// CbcAttendancePeriod represents a single row in cbc_attendance_periods.
type CbcAttendancePeriod struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	SchoolID       string `json:"school_id"`
	AcademicTermID string `json:"academic_term_id"`
	ClassID        string `json:"class_id"`
	LearningAreaID string `json:"cbc_learning_area_id"`
	DateRecorded   string `json:"date_recorded"`
	RecordedBy     string `json:"recorded_by"`
	CreatedAt      string `json:"created_at"`
}

// CbcAttendanceLog represents a single row in cbc_attendance_logs.
type CbcAttendanceLog struct {
	ID         string  `json:"id"`
	TenantID   string  `json:"tenant_id"`
	PeriodID   string  `json:"cbc_attendance_period_id"`
	StudentID  string  `json:"student_id"`
	Status     string  `json:"status"`
	Remarks    *string `json:"remarks"`
	RecordedBy string  `json:"recorded_by"`
}

// AttendancePeriodSummary is the enriched period view for the list/register.
type AttendancePeriodSummary struct {
	ID               string `json:"id"`
	DateRecorded     string `json:"date_recorded"`
	LearningAreaID   string `json:"cbc_learning_area_id"`
	LearningAreaName string `json:"learning_area_name"`
	RecordedByName   string `json:"recorded_by_name"`
	RecordedByID     string `json:"recorded_by_id"`
	RecordedAt       string `json:"recorded_at"`
	TotalStudents    int    `json:"total_students"`
	PresentCount     int    `json:"present_count"`
	AbsentCount      int    `json:"absent_count"`
	LateCount        int    `json:"late_count"`
	ExcusedCount     int    `json:"excused_count"`
	UnmarkedCount    int    `json:"unmarked_count"`
}

// AttendanceLogDetail is a log entry with recorder details.
type AttendanceLogDetail struct {
	CbcAttendanceLog
	RecorderFirstName string `json:"recorder_first_name"`
	RecorderLastName  string `json:"recorder_last_name"`
	RecordedByLabel   string `json:"recorded_by_label"`
}

// AttendanceHeatmapDay is a single day cell in the term-level heatmap.
type AttendanceHeatmapDay struct {
	Date        string   `json:"date"`
	PeriodCount int      `json:"period_count"`
	PresentRate *float64 `json:"present_rate"`
	TotalMarks  int      `json:"total_marks"`
}

// AttendanceGap is a timetable slot that has no corresponding attendance period.
type AttendanceGap struct {
	SlotID           string  `json:"slot_id"`
	ClassID          string  `json:"class_id"`
	LearningAreaID   *string `json:"cbc_learning_area_id"`
	LearningAreaName string  `json:"learning_area_name"`
	DayOfWeek        int     `json:"day_of_week"`
	StartTime        string  `json:"start_time"`
	EndTime          string  `json:"end_time"`
	Date             string  `json:"date"`
}

// CreatePeriodRequest is the request body for POST attendance/periods.
type CreatePeriodRequest struct {
	LearningAreaID string `json:"cbc_learning_area_id"`
	DateRecorded   string `json:"date_recorded"`
}

// SaveLogRequest is the request body for POST attendance/logs.
type SaveLogRequest struct {
	PeriodID  string  `json:"cbc_attendance_period_id"`
	StudentID string  `json:"student_id"`
	Status    string  `json:"status"`
	Remarks   *string `json:"remarks"`
}

// BatchSaveLogsRequest is the request body for POST attendance/logs/batch.
type BatchSaveLogsRequest struct {
	PeriodID string         `json:"cbc_attendance_period_id"`
	Marks    []BatchLogMark `json:"marks"`
}

// BatchLogMark is a single entry in a batch save.
type BatchLogMark struct {
	StudentID string  `json:"student_id"`
	Status    string  `json:"status"`
	Remarks   *string `json:"remarks"`
}
