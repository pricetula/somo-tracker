/**
 * CBC Attendance API client.
 *
 * Endpoints:
 *   GET    /api/v1/cbc/classes/:classId/attendance/periods?date=:date
 *          → CbcAttendancePeriod[]
 *   GET    /api/v1/cbc/classes/:classId/attendance/periods?from=:from&to=:to
 *          → AttendancePeriodSummary[]
 *   POST   /api/v1/cbc/classes/:classId/attendance/periods  → CbcAttendancePeriod
 *   GET    /api/v1/cbc/attendance/periods/:periodId/logs    → CbcAttendanceLogDetail[]
 *   PUT    /api/v1/cbc/attendance/logs/:logId               → CbcAttendanceLog
 *   POST   /api/v1/cbc/attendance/logs                      → CbcAttendanceLog
 *   POST   /api/v1/cbc/attendance/logs/batch                → CbcAttendanceLog[]
 *   GET    /api/v1/cbc/attendance/periods/:periodId         → AttendancePeriodSummary
 *   GET    /api/v1/cbc/classes/:classId/attendance/heatmap?term_id=:termId  → AttendanceHeatmapDay[]
 *   GET    /api/v1/cbc/classes/:classId/attendance/gaps?from=:from&to=:to   → AttendanceGap[]
 *   GET    /api/v1/cbc/attendance/slots/today?teacher_id=:id → DashboardSlotCard[]
 */

import { api } from "@/lib/api/client";
import type {
    CbcAttendancePeriod,
    CbcAttendanceLog,
    CbcAttendanceLogDetail,
    AttendanceStudentRow,
    AttendanceStatus,
    AttendancePeriodSummary,
    AttendanceHeatmapDay,
    AttendanceGap,
    DashboardSlotCard,
} from "@/features/cbc/types";

// ─── Fetch attendance periods for a class on a given date ─────────────────

export async function fetchCbcAttendancePeriods(
    classId: string,
    date: string
): Promise<CbcAttendancePeriod[]> {
    return api.get<CbcAttendancePeriod[]>(
        `/api/v1/cbc/classes/${classId}/attendance/periods?date=${date}`
    );
}

// ─── Fetch attendance period summaries for a date range (list view) ─────────

export async function fetchCbcAttendancePeriodSummaries(
    classId: string,
    from: string,
    to: string
): Promise<AttendancePeriodSummary[]> {
    return api.get<AttendancePeriodSummary[]>(
        `/api/v1/cbc/classes/${classId}/attendance/periods?from=${from}&to=${to}`
    );
}

// ─── Fetch a single period summary ────────────────────────────────────────

export async function fetchCbcAttendancePeriodDetail(
    periodId: string
): Promise<AttendancePeriodSummary> {
    return api.get<AttendancePeriodSummary>(`/api/v1/cbc/attendance/periods/${periodId}`);
}

// ─── Create an attendance period (start taking attendance) ─────────────────

export async function createCbcAttendancePeriod(
    classId: string,
    cbcLearningAreaId: string,
    date: string
): Promise<CbcAttendancePeriod> {
    return api.post<CbcAttendancePeriod>(`/api/v1/cbc/classes/${classId}/attendance/periods`, {
        cbc_learning_area_id: cbcLearningAreaId,
        date_recorded: date,
    });
}

// ─── Fetch all attendance logs for a period (with recorder details) ───────

export async function fetchCbcAttendanceLogs(periodId: string): Promise<CbcAttendanceLogDetail[]> {
    return api.get<CbcAttendanceLogDetail[]>(`/api/v1/cbc/attendance/periods/${periodId}/logs`);
}

// ─── Fetch attendance heatmap data for a term ─────────────────────────────

export async function fetchCbcAttendanceHeatmap(
    classId: string,
    termId: string
): Promise<AttendanceHeatmapDay[]> {
    return api.get<AttendanceHeatmapDay[]>(
        `/api/v1/cbc/classes/${classId}/attendance/heatmap?term_id=${termId}`
    );
}

// ─── Fetch attendance gaps for a date range ───────────────────────────────

export async function fetchCbcAttendanceGaps(
    classId: string,
    from: string,
    to: string
): Promise<AttendanceGap[]> {
    return api.get<AttendanceGap[]>(
        `/api/v1/cbc/classes/${classId}/attendance/gaps?from=${from}&to=${to}`
    );
}

// ─── Record a single attendance mark ──────────────────────────────────────

/** Creates a new log or updates an existing one for this student+period. */
export async function saveAttendanceMark(
    periodId: string,
    studentId: string,
    status: AttendanceStatus,
    remarks?: string
): Promise<CbcAttendanceLog> {
    return api.post<CbcAttendanceLog>(`/api/v1/cbc/attendance/logs`, {
        cbc_attendance_period_id: periodId,
        student_id: studentId,
        status,
        remarks: remarks ?? null,
    });
}

// ─── Batch save attendance marks (save all at once) ───────────────────────

export async function saveAttendanceBatch(
    periodId: string,
    marks: Array<{
        student_id: string;
        status: AttendanceStatus;
        remarks?: string;
    }>
): Promise<CbcAttendanceLog[]> {
    return api.post<CbcAttendanceLog[]>(`/api/v1/cbc/attendance/logs/batch`, {
        cbc_attendance_period_id: periodId,
        marks,
    });
}

// ─── Mark all remaining students as Present ───────────────────────────────

export async function markRemainingAsPresent(
    periodId: string,
    studentIds: string[]
): Promise<CbcAttendanceLog[]> {
    return api.post<CbcAttendanceLog[]>(`/api/v1/cbc/attendance/logs/batch`, {
        cbc_attendance_period_id: periodId,
        marks: studentIds.map((student_id) => ({
            student_id,
            status: "PRESENT" as AttendanceStatus,
        })),
    });
}

// ─── Fetch today's slots for a teacher (My Attendance dashboard) ──────────

export async function fetchTeacherTodaySlots(teacherId: string): Promise<DashboardSlotCard[]> {
    return api.get<DashboardSlotCard[]>(
        `/api/v1/cbc/attendance/slots/today?teacher_id=${teacherId}`
    );
}

// ─── Fetch enrolled students for a class (for attendance grid) ────────────

export async function fetchClassStudents(
    classId: string,
    academicTermId: string
): Promise<AttendanceStudentRow[]> {
    return api.get<AttendanceStudentRow[]>(
        `/api/v1/cbc/classes/${classId}/students?academic_term_id=${academicTermId}`
    );
}
