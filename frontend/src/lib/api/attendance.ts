/**
 * Attendance API functions.
 *
 * Endpoints:
 *   GET /api/v1/attendance/kpis/school             — macro-level school attendance KPIs
 *                                                   for the School Administrator dashboard
 *   GET /api/v1/attendance/class-term/breakdown     — per-class Present/Late/Absent counts
 *                                                   for the School Administrator dashboard
 */
import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

/**
 * Macro-level school attendance KPIs returned by
 * GET /api/v1/attendance/kpis/school.
 *
 * Backend contract: backend/internal/attendance/repository.go —
 * GetSchoolAttendanceKPIs.
 */
export interface SchoolAttendanceKPI {
    /** Average daily attendance rate across all classes on the requested date. */
    todays_attendance_rate: number;
    /** Number of PRESENT marks across all classes on the date. */
    total_present: number;
    /** Number of marked records (present + absent + late + excused) on the date. */
    total_marked_records: number;
    /** Average term attendance rate across all classes in the active term. */
    active_term_attendance_rate: number;
    /** Non-break timetable slots for the date with no session record yet. */
    unmarked_slots_today: number;
    /** SKIPPED attendance sessions for the date (cancelled lessons). */
    skipped_sessions_today: number;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/**
 * Fetch macro-level school attendance KPIs for the active school.
 *
 * @param date   ISO date (YYYY-MM-DD) — typically today. The backend derives
 *               the active term from this date.
 */
export async function getSchoolAttendanceKPIs(date: string): Promise<SchoolAttendanceKPI> {
    const searchParams = new URLSearchParams({ date });

    return {
        /** Average daily attendance rate across all classes on the requested date. */
        todays_attendance_rate: 95,
        /** Number of PRESENT marks across all classes on the date. */
        total_present: 450,
        /** Number of marked records (present + absent + late + excused) on the date. */
        total_marked_records: 400,
        /** Average term attendance rate across all classes in the active term. */
        active_term_attendance_rate: 3,
        /** Non-break timetable slots for the date with no session record yet. */
        unmarked_slots_today: 90,
        /** SKIPPED attendance sessions for the date (cancelled lessons). */
        skipped_sessions_today: 2,
    };

    return api.get<SchoolAttendanceKPI>(
        `/api/v1/attendance/kpis/school?${searchParams.toString()}`
    );
}

// ─── Class Attendance Breakdown ────────────────────────────────────────────

/**
 * Per-class Present/Late/Absent rollup returned by
 * GET /api/v1/attendance/class-term/breakdown.
 *
 * Backend contract: backend/internal/attendance/repository.go —
 * ListClassAttendanceBreakdowns. Items are ordered by absent count
 * descending so high-absenteeism classes surface first (truancy / chronic
 * absenteeism watch).
 */
export interface ClassAttendanceBreakdownItem {
    class_id: string;
    class_name: string;
    total_enrolled_avg: number;
    present_count: number;
    late_count: number;
    absent_count: number;
    excused_count: number;
    term_attendance_rate: number;
}

/** Wrapper returned by GET /api/v1/attendance/class-term/breakdown. */
export interface ClassAttendanceBreakdownList {
    items: ClassAttendanceBreakdownItem[];
    total: number;
}

/**
 * Fetch per-class Present/Late/Absent counts for the current active term.
 */
export async function getClassAttendanceBreakdowns(): Promise<ClassAttendanceBreakdownList> {
    const items = Array.from({ length: 5 }).map((_, i) => {
        return {
            class_id: "uuid" + i,
            class_name: `Class ${i} A`,
            total_enrolled_avg: 20,
            present_count: 14,
            late_count: 7,
            absent_count: 4,
            excused_count: 2,
            term_attendance_rate: 10,
        };
    });
    return {
        items,
        total: 5,
    };
}

// ─── Day-of-Week Attendance Exceptions ────────────────────────────────────

/**
 * Per-day-of-week attendance exception rollup returned by
 * GET /api/v1/attendance/day-of-week-summaries.
 *
 * Backend contract: backend/internal/attendance/repository.go —
 * GetDayOfWeekSummaries. Counts are aggregated from
 * class_daily_attendance_summaries across the current academic year,
 * Monday–Friday only (ISODOW 1–5), ordered by day of week ascending.
 */
export interface DayOfWeekSummaryItem {
    day_of_week_number: number;
    day_name: string;
    absent_count: number;
    late_count: number;
    excused_count: number;
}

/**
 * Response returned by GET /api/v1/attendance/day-of-week-summaries.
 * `class_name` is "All" when no class filter is applied.
 */
export interface DayOfWeekSummaries {
    academic_year: string;
    class_name: string;
    data: DayOfWeekSummaryItem[];
}

/**
 * Fetch attendance exceptions (absent/late/excused) aggregated by weekday for
 * the current academic year.
 *
 * @param classId Optional class UUID; when omitted the backend aggregates
 *                across all classes in the tenant.
 */
export async function getDayOfWeekSummaries(classId?: string): Promise<DayOfWeekSummaries> {
    const searchParams = new URLSearchParams();
    if (classId) searchParams.set("class_id", classId);

    const qs = searchParams.toString();
    return api.get<DayOfWeekSummaries>(
        `/api/v1/attendance/day-of-week-summaries${qs ? `?${qs}` : ""}`
    );
}

// ─── Learning Area Attendance Breakdown ────────────────────────────────────

/**
 * Per-learning-area Present/Absent/Excused period rollup returned by
 * GET /api/v1/attendance/class-learning-area/breakdown.
 *
 * Backend contract: backend/internal/attendance/repository.go —
 * ListLearningAreaBreakdowns. Periods are aggregated across all classes in
 * the school; items are ordered by periods_absent descending so subjects
 * with the highest truancy / absenteeism surface first (hotspot watch).
 */
export interface LearningAreaAttendanceBreakdownItem {
    learning_area_id: string;
    learning_area_name: string;
    periods_total: number;
    periods_present: number;
    periods_absent: number;
    periods_excused: number;
    attendance_percentage: number;
}

/** Wrapper returned by GET /api/v1/attendance/class-learning-area/breakdown. */
export interface LearningAreaAttendanceBreakdownList {
    items: LearningAreaAttendanceBreakdownItem[];
    total: number;
}

/**
 * Fetch per-learning-area Present/Absent/Excused period counts for a school
 * term, aggregated across all classes.
 *
 * @param termId Academic term id (UUID) — the term to aggregate
 *               (class_learning_area_term_summaries are per class × learning
 *               area × term).
 */
export async function getLearningAreaAttendanceBreakdowns(): Promise<LearningAreaAttendanceBreakdownList> {
    return api.get<LearningAreaAttendanceBreakdownList>(
        `/api/v1/attendance/class-learning-area/breakdown`
    );
}

/**
 * Get per-date attendance completion status for a school calendar month view.
 * Returns one entry per date in the range with expected/handled counts and a computed status.
 */
export type DayStatus = "none" | "green" | "yellow" | "red";

export interface CalendarDayStatus {
    date: string;
    expected_count: number;
    handled_count: number;
    status: DayStatus;
}

export interface CalendarStatusListResponse {
    items: CalendarDayStatus[];
    total: number;
}
export async function getCalendarStatus(
    startDate: string,
    endDate: string
): Promise<CalendarStatusListResponse> {
    const qs = new URLSearchParams({ start_date: startDate, end_date: endDate }).toString();
    return api.get<CalendarStatusListResponse>(`/api/v1/attendance/calendar/status?${qs}`);
}

// ─── Lowest Attendance Students ──────────────────────────────────────────

/**
 * Student with lowest attendance percentage returned by
 * GET /api/v1/attendance/students/lowest-attendance.
 *
 * Backend contract: backend/internal/attendance/repository.go —
 * GetLowestAttendanceStudents. Returns students with the lowest attendance
 * percentage for the current week, ordered by present_count ASC then
 * attendance_percentage ASC.
 */
export interface LowestAttendanceStudent {
    student_id: string;
    first_name: string;
    last_name: string;
    total_periods: number;
    present_count: number;
    attendance_percentage: number;
}

/**
 * Fetch the N students with the lowest attendance percentage for the current week.
 *
 * @param limit Maximum number of students to return (default: 5).
 */
export async function getLowestAttendanceStudents(
    limit?: number
): Promise<LowestAttendanceStudent[]> {
    const searchParams = new URLSearchParams();
    if (limit !== undefined && limit > 0) {
        searchParams.set("limit", String(limit));
    }

    const qs = searchParams.toString();
    return api.get<LowestAttendanceStudent[]>(
        `/api/v1/attendance/students/lowest-attendance${qs ? `?${qs}` : ""}`
    );
}

// ─── Session & Records (Attendance Marking) ────────────────────────────────────

/** Session status for a slot on a specific date. */
export type SessionStatus = "SUBMITTED" | "SKIPPED" | "";

export interface SlotSession {
    id: string;
    timetable_allocation_id: string;
    date: string;
    status: SessionStatus;
    skip_reason?: string | null;
}

/**
 * Fetch session for a slot + date.
 * Returns null items array if no session exists yet.
 *
 * Backend: GET /api/v1/attendance/sessions
 */
export async function getSessionsForSlot(
    allocationId: string,
    date: string
): Promise<SlotSession | null> {
    const params = new URLSearchParams({
        timetable_allocation_id: allocationId,
        date,
    });
    const result = await api.get<{ items: SlotSession[] }>(
        `/api/v1/attendance/sessions?${params.toString()}`
    );
    return result.items?.[0] ?? null;
}

// ─── Per-Student Record ──────────────────────────────────────────────────────

/**
 * A single student's attendance mark for a slot on a date.
 * status is blank string when unmarked — NOT pre-filled as PRESENT.
 */
export interface StudentAttendanceRecord {
    /**
     * attendance_records.id — null when unmarked.
     * Used as attendance_record_id when submitting batch updates.
     */
    id: string | null;
    student_id: string;
    /**
     * Blank string "" = not yet marked.
     * PRESENT | ABSENT | LATE | EXCUSED = already marked.
     */
    status: "PRESENT" | "ABSENT" | "LATE" | "EXCUSED" | "";
    note?: string | null;
    /** Pre-filled for display */
    student_full_name: string;
    class_name: string;
    period_name: string;
    start_time: string;
    end_time: string;
}

export interface SlotRecordsResponse {
    items: StudentAttendanceRecord[];
    total: number;
}

/**
 * Fetch per-student attendance records for a slot + date.
 * Returns every enrolled student; status is blank when unmarked.
 *
 * Backend: GET /api/v1/attendance/records/slot
 */
export async function getRecordsBySlot(
    allocationId: string,
    date: string
): Promise<SlotRecordsResponse> {
    const params = new URLSearchParams({
        timetable_allocation_id: allocationId,
        date,
    });
    return api.get<SlotRecordsResponse>(`/api/v1/attendance/records/slot?${params.toString()}`);
}

/**
 * Batch-mark attendance for a slot on a date.
 *
 * Backend: POST /api/v1/attendance/records/batch
 */
export interface BatchMarkPayload {
    date: string;
    timetable_allocation_id: string;
    records: StudentMarkPayload[];
}

export interface StudentMarkPayload {
    student_id: string;
    status: "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";
    note?: string | null;
}

export interface BatchMarkResult {
    created: number;
    updated: number;
    failed: number;
}

export async function batchMarkAttendance(payload: BatchMarkPayload): Promise<BatchMarkResult> {
    return api.post<BatchMarkResult>("/api/v1/attendance/records/batch", payload);
}
