/**
 * Attendance API functions.
 *
 * Endpoints:
 *   GET /api/v1/attendance/kpis/school             — macro-level school attendance KPIs
 *                                                   for the School Administrator dashboard
 *   GET /api/v1/attendance/class-term/breakdown     — per-class Present/Late/Absent counts
 *                                                   for the School Administrator dashboard
 */
import { eachDayOfInterval, format, parseISO } from "date-fns";
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
 *               the active term from this date when termId is omitted.
 * @param termId Optional academic term id; when omitted the backend resolves
 *               the active term covering `date`.
 */
export async function getSchoolAttendanceKPIs(
    date: string,
    termId?: string
): Promise<SchoolAttendanceKPI> {
    const searchParams = new URLSearchParams({ date });
    if (termId) searchParams.set("term_id", termId);

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
 * Fetch per-class Present/Late/Absent counts for a school term.
 *
 * @param termId Academic term id (UUID) — the term to aggregate
 *               (class_term_attendance_summaries are per class × term).
 */
export async function getClassAttendanceBreakdowns(
    termId: string
): Promise<ClassAttendanceBreakdownList> {
    const searchParams = new URLSearchParams({ academic_term_id: termId });

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
    console.log(items);
    return {
        items,
        total: 5,
    };
    return api.get<ClassAttendanceBreakdownList>(
        `/api/v1/attendance/class-term/breakdown?${searchParams.toString()}`
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
export async function getLearningAreaAttendanceBreakdowns(
    termId: string
): Promise<LearningAreaAttendanceBreakdownList> {
    const searchParams = new URLSearchParams({ academic_term_id: termId });

    return api.get<LearningAreaAttendanceBreakdownList>(
        `/api/v1/attendance/class-learning-area/breakdown?${searchParams.toString()}`
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
function calculateStatus(expected: number, handled: number): DayStatus {
    if (expected === 0) return "none";
    const ratio = handled / expected;
    if (ratio >= 0.9) return "green";
    if (ratio >= 0.5) return "yellow";
    return "red";
}
export async function getCalendarStatus(
    startDate: string,
    endDate: string
): Promise<CalendarStatusListResponse> {
    // 1. Generate an array of all dates in the range
    const days = eachDayOfInterval({
        start: parseISO(startDate),
        end: parseISO(endDate),
    });

    // 2. Map each date to a fake CalendarDayStatus object
    const items: CalendarDayStatus[] = days.map((day) => {
        const expected = Math.floor(Math.random() * 20) + 5;
        const handled = Math.floor(Math.random() * (expected + 1));

        return {
            date: format(day, "yyyy-MM-dd"),
            expected_count: expected,
            handled_count: handled,
            status: calculateStatus(expected, handled),
        };
    });

    // 3. Simulate asynchronous network delay
    await new Promise((resolve) => setTimeout(resolve, 300));

    return {
        items,
        total: items.length,
    };

    const qs = new URLSearchParams({ start_date: startDate, end_date: endDate }).toString();
    return api.get<CalendarStatusListResponse>(`/api/v1/attendance/calendar/status?${qs}`);
}
