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

    return api.get<ClassAttendanceBreakdownList>(
        `/api/v1/attendance/class-term/breakdown?${searchParams.toString()}`
    );
}
