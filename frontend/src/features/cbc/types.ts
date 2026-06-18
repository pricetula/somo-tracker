// ─── Reference ──────────────────────────────────────────────────────────────
//
// All types map directly to backend Go JSON responses. Field names use
// snake_case to match Go's json tags 1:1 — no camelCase conversion.
//
// cbc_timetable_slots DB schema:
//   id, tenant_id, school_id, academic_year_id, class_id, teacher_id,
//   cbc_learning_area_id (nullable), room_identifier (nullable),
//   day_of_week (1-7), start_time (TIME), end_time (TIME)
//
// Hard DB constraints:
//   excl_cbc_timetable_teacher — GiST EXCLUDE on teacher_id + timerange
//   excl_cbc_timetable_room    — GiST EXCLUDE on room_identifier + timerange
//   day_of_week CHECK 1..7
//   cbc_learning_area_id nullable
// ─────────────────────────────────────────────────────────────────────────────

// ═══════════════════════════════════════════════════════════════════════════
// TIMETABLE BUILDER
// ═══════════════════════════════════════════════════════════════════════════

/** A single timetable slot — matches backend TimetableSlot 1:1. */
export interface CbcTimetableSlot {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_year_id: string;
    class_id: string;
    teacher_id: string;
    cbc_learning_area_id: string | null;
    room_identifier: string | null;
    day_of_week: number; // 1=Mon … 7=Sun
    start_time: string; // "HH:mm" returned by backend
    end_time: string; // "HH:mm"
}

/** Payload for POST /api/v1/cbc/classes/:classId/timetable. */
export interface CbcTimetableSlotCreatePayload {
    class_id: string;
    teacher_id: string;
    cbc_learning_area_id: string | null;
    room_identifier: string | null;
    day_of_week: number;
    start_time: string;
    end_time: string;
}

/** Payload for PUT /api/v1/cbc/timetable/:slotId. */
export interface CbcTimetableSlotUpdatePayload {
    id: string;
    teacher_id?: string;
    cbc_learning_area_id?: string | null;
    room_identifier?: string | null;
    day_of_week?: number;
    start_time?: string;
    end_time?: string;
}

/** Conflict pre-check result — matches backend ConflictError JSON. */
export interface CbcSlotConflict {
    type: "teacher" | "room";
    entity: string; // teacher name or room identifier
    class_name: string; // colliding class name
    day_of_week: number;
    start_time: string;
    end_time: string;
}

/** Operating day — matches handler's dayObj JSON. */
export interface OperatingDay {
    value: number; // 1-7
    label: string; // "Monday", "Tuesday", etc.
    short_label: string; // "Mon", "Tue", etc.
}

/** A learning area option — matches backend LearningAreaBrief. */
export interface LearningAreaOption {
    id: string;
    name: string;
    code: string;
    grade_id?: string; // only present when explicitly included
}

/** A teacher option — matches backend TeacherBrief. */
export interface TeacherOption {
    id: string;
    name: string; // "First Last"
    first_name: string;
    last_name: string;
    email: string;
}

/** Request body for POST /api/v1/cbc/timetable/duplicate-day. */
export interface DuplicateDayPayload {
    source_day: number;
    target_days: number[];
    academic_year_id: string;
    class_id: string;
}

/** Request body for POST /api/v1/cbc/timetable/copy-from-class. */
export interface CopyFromClassPayload {
    source_class_id: string;
    academic_year_id: string;
    target_class_id: string;
}

/** A skipped-slot entry — matches backend SlotSkipReason. */
export interface SlotSkipReason {
    day_of_week: number;
    start_time: string;
    reason: string;
}

/** Bulk operation result — matches backend BulkOperationResult. */
export interface BulkOperationResult {
    total_copied: number;
    skipped: SlotSkipReason[];
}

/** Attendance count — matches backend AttendanceCount. */
export interface AttendanceCount {
    count: number;
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE
// ═══════════════════════════════════════════════════════════════════════════

export type AttendanceStatus = "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";

/** A single attendance period — matches backend CbcAttendancePeriod. */
export interface CbcAttendancePeriod {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_term_id: string;
    class_id: string;
    cbc_learning_area_id: string;
    date_recorded: string;
}

/** A single attendance log entry — matches backend CbcAttendanceLog. */
export interface CbcAttendanceLog {
    id: string;
    tenant_id: string;
    cbc_attendance_period_id: string;
    student_id: string;
    status: AttendanceStatus;
    remarks: string | null;
    recorded_by: string;
}

/**
 * A student row in the attendance grid.
 * Composite of backend StudentAttendanceRow + merged log status.
 * `status` and `log_id` are populated by merging the period's logs.
 * `syncPending` is a client-only field for optimistic UI.
 */
export interface AttendanceStudentRow {
    student_id: string;
    student_name: string; // "First Last"
    first_name: string;
    last_name: string;
    admission_number?: string;
    status: AttendanceStatus | null; // null = not yet recorded; merged from logs
    log_id: string | null; // merged from logs
    syncPending?: boolean; // client-only: optimistic save in flight
}

/** A learning area slot shown on the teacher dashboard — matches backend SlotBrief. */
export interface AttendanceSlotColumn {
    period_id: string;
    learning_area_name: string;
    start_time: string;
    end_time: string;
}

/** Offline queue entry for optimistic attendance saves (client-only). */
export interface OfflineAttendanceEntry {
    localId: string;
    periodId: string;
    studentId: string;
    status: AttendanceStatus;
    timestamp: number;
    retryCount: number;
}

/**
 * Attendance period summary for the period list view.
 * Includes recorder info and per-status breakdown counts.
 */
export interface AttendancePeriodSummary {
    id: string;
    date_recorded: string;
    cbc_learning_area_id: string;
    learning_area_name: string;
    recorded_by_name: string;
    recorded_by_id: string;
    recorded_at: string; // ISO timestamp when period was created
    total_students: number;
    present_count: number;
    absent_count: number;
    late_count: number;
    excused_count: number;
    unmarked_count: number;
}

/** A single day cell in the term-level heatmap. */
export interface AttendanceHeatmapDay {
    date: string; // "YYYY-MM-DD"
    period_count: number;
    present_rate: number | null; // null = no periods at all (empty/gray)
    total_marks: number;
}

/** A timetable slot that has no corresponding attendance period (a gap). */
export interface AttendanceGap {
    slot_id: string;
    class_id: string;
    cbc_learning_area_id: string | null;
    learning_area_name: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
    date: string; // specific date this slot falls on
}

/**
 * Enriched attendance log with recorder details for the register view.
 */
export interface CbcAttendanceLogDetail extends CbcAttendanceLog {
    recorder_first_name: string;
    recorder_last_name: string;
    /** Human-friendly "Marked by X, HH:MMam" */
    recorded_by_label: string;
}

/** Status of a dashboard slot card. */
export type SlotCardStatus = "ongoing" | "done" | "incomplete" | "upcoming" | "past_not_taken";

/**
 * Enriched slot card for the teacher dashboard (My Attendance).
 * Extends the bare slot with attendance status and completeness info.
 */
export interface DashboardSlotCard {
    slot_id: string;
    class_id: string;
    class_name: string;
    learning_area_name: string;
    learning_area_id: string;
    start_time: string;
    end_time: string;
    day_of_week: number;
    /** The matching attendance period, if one exists. */
    attendance_period_id: string | null;
    status: SlotCardStatus;
    total_students: number;
    marked_count: number;
    /** Whether the logged-in teacher is the usual teacher for this slot. */
    is_usual_teacher: boolean;
    /** Academic term ID for this date */
    academic_term_id: string;
}

// ═══════════════════════════════════════════════════════════════════════════
// SHARED REFERENCE DATA
// ═══════════════════════════════════════════════════════════════════════════

export interface CbcLearningArea {
    id: string;
    tenant_id: string;
    school_id: string;
    education_system_id: string;
    grade_id: string;
    name: string;
    code: string;
}

export interface CbcClassTeacher {
    id: string;
    class_id: string;
    user_id: string;
    learning_area_id: string;
    is_primary: boolean;
    teacher_name?: string;
}

export interface UserBrief {
    id: string;
    first_name: string;
    last_name: string;
    email: string;
}

/** Class detail for the page header. */
export interface ClassDetail {
    id: string;
    name: string;
    stream: string;
    grade_name: string;
    education_system_name: string;
    academic_year_name: string;
    teacher_count: number;
    student_count: number;
}
