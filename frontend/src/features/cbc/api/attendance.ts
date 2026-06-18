/**
 * CBC Attendance API client.
 *
 * Endpoints (to be implemented in the Go backend):
 *   GET    /api/v1/cbc/classes/:classId/attendance/periods?date=:date
 *          → CbcAttendancePeriod[]
 *   POST   /api/v1/cbc/classes/:classId/attendance/periods  → CbcAttendancePeriod
 *   GET    /api/v1/cbc/attendance/periods/:periodId/logs    → CbcAttendanceLog[]
 *   PUT    /api/v1/cbc/attendance/logs/:logId               → CbcAttendanceLog
 *   POST   /api/v1/cbc/attendance/logs/batch                → CbcAttendanceLog[]
 *   GET    /api/v1/cbc/attendance/slots/today?teacher_id=:id → AttendanceSlotColumn[]
 */

import { api } from "@/lib/api/client";
import type {
    CbcAttendancePeriod,
    CbcAttendanceLog,
    AttendanceStudentRow,
    AttendanceSlotColumn,
    AttendanceStatus,
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

// ─── Fetch all attendance logs for a period ───────────────────────────────

export async function fetchCbcAttendanceLogs(periodId: string): Promise<CbcAttendanceLog[]> {
    return api.get<CbcAttendanceLog[]>(`/api/v1/cbc/attendance/periods/${periodId}/logs`);
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

// ─── Fetch today's slots for a teacher (My Attendance dashboard) ──────────

export async function fetchTeacherTodaySlots(teacherId: string): Promise<AttendanceSlotColumn[]> {
    return api.get<AttendanceSlotColumn[]>(
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
