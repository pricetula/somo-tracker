/**
 * TypeScript interfaces for the Attendance feature.
 *
 * Maps to backend/internal/attendance/domain.go
 */

// ─── Enums ────────────────────────────────────────────────────────────────

export type AttendanceStatus = "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";

export type SessionStatus = "SUBMITTED" | "SKIPPED";

// ─── Domain Models ────────────────────────────────────────────────────────

export interface AttendanceRecord {
    id: string;
    tenant_id: string;
    school_id: string;
    student_id: string;
    timetable_slot_id: string;
    academic_term_id: string;
    date: string;
    status: AttendanceStatus;
    marked_by: string;
    note?: string | null;
    attendance_session_id?: string | null;
    created_at: string;
    updated_at: string;
}

export interface AttendanceSession {
    id: string;
    tenant_id: string;
    school_id: string;
    timetable_slot_id: string;
    date: string;
    status: SessionStatus;
    skip_reason?: string | null;
    created_at: string;
    updated_at: string;
}

export interface SessionWithEnrichedData extends AttendanceSession {
    class_name: string;
    stream_name?: string;
    grade_level: string;
    period_name: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
    learning_area_id: string;
    learning_area_name?: string | null;
    teacher_name?: string | null;
}

export interface RecordWithEnrichedData extends AttendanceRecord {
    student_full_name: string;
    class_name: string;
    grade_level: string;
    stream_name?: string;
    period_name: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
    learning_area_id: string;
    learning_area_name?: string | null;
}

export interface AttendanceTermSummary {
    id: string;
    tenant_id: string;
    school_id: string;
    student_id: string;
    academic_term_id: string;
    learning_area_id: string;
    learning_area_name?: string;
    periods_total: number;
    periods_present: number;
    periods_absent: number;
    periods_late: number;
    periods_excused: number;
    attendance_percentage: number;
    last_refreshed_at: string;
    updated_at: string;
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface SessionListResponse {
    items: SessionWithEnrichedData[];
    total: number;
}

export interface RecordListResponse {
    items: RecordWithEnrichedData[];
    total: number;
}

export interface SummaryListResponse {
    items: AttendanceTermSummary[];
    total: number;
}

export interface RefreshSummaryResponse {
    message: string;
    term_id: string;
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface CreateSessionPayload {
    timetable_slot_id: string;
    date: string;
    status: SessionStatus;
    skip_reason?: string | null;
}

export interface UpdateSessionPayload {
    status?: SessionStatus;
    skip_reason?: string | null;
}

export interface StudentAttendanceMark {
    student_id: string;
    status: AttendanceStatus;
    note?: string | null;
}

export interface BatchMarkPayload {
    date: string;
    timetable_slot_id: string;
    records: StudentAttendanceMark[];
}

export interface BatchMarkResult {
    created: number;
    updated: number;
    failed: number;
}

export interface UpdateRecordPayload {
    status?: AttendanceStatus;
    note?: string | null;
}
