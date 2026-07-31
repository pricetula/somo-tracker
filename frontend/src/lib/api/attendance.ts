/**
 * Attendance API functions.
 *
 * Endpoints (from backend/internal/attendance/handler.go):
 *   Sessions:
 *     POST   /api/v1/attendance/sessions
 *     GET    /api/v1/attendance/sessions
 *     GET    /api/v1/attendance/sessions/:id
 *     PUT    /api/v1/attendance/sessions/:id
 *     GET    /api/v1/attendance/sessions/class/:class_id/date/:date
 *
 *   Records:
 *     POST   /api/v1/attendance/records/batch
 *     GET    /api/v1/attendance/records/slot?timetable_slot_id=&date=
 *     GET    /api/v1/attendance/records/student/:student_id?term_id=
 *     GET    /api/v1/attendance/records/class/:class_id/date/:date?term_id=
 *     GET    /api/v1/attendance/records
 *     PUT    /api/v1/attendance/records/:id
 *
 *   Summaries:
 *     GET    /api/v1/attendance/summaries/student/:student_id?term_id=
 *     GET    /api/v1/attendance/summaries/class/:class_id?term_id=
 *     POST   /api/v1/attendance/summaries/refresh
 */

import { api } from "./client";

import type {
    SessionWithEnrichedData,
    SessionListResponse,
    CreateSessionPayload,
    UpdateSessionPayload,
    RecordWithEnrichedData,
    RecordListResponse,
    BatchMarkPayload,
    BatchMarkResult,
    UpdateRecordPayload,
    AttendanceTermSummary,
    RefreshSummaryResponse,
    CalendarStatusListResponse,
} from "@/features/attendance/types";

// ─── Sessions ─────────────────────────────────────────────────────────────

/** Create a new attendance session. */
export async function createSession(data: CreateSessionPayload): Promise<SessionWithEnrichedData> {
    return api.post<SessionWithEnrichedData>("/api/v1/attendance/sessions", data);
}

/** List attendance sessions with optional filters. */
export async function listSessions(
    params: {
        timetable_slot_id?: string;
        date?: string;
        status?: string;
        class_id?: string;
    } = {}
): Promise<SessionListResponse> {
    const searchParams = new URLSearchParams();
    if (params.timetable_slot_id) searchParams.set("timetable_slot_id", params.timetable_slot_id);
    if (params.date) searchParams.set("date", params.date);
    if (params.status) searchParams.set("status", params.status);
    if (params.class_id) searchParams.set("class_id", params.class_id);
    const qs = searchParams.toString();
    return api.get<SessionListResponse>(`/api/v1/attendance/sessions?${qs}`);
}

/** Get a single attendance session by ID. */
export async function getSession(id: string): Promise<SessionWithEnrichedData> {
    return api.get<SessionWithEnrichedData>(`/api/v1/attendance/sessions/${id}`);
}

/** Get sessions for a class on a specific date. */
export async function getSessionsForClassDate(
    classId: string,
    date: string
): Promise<SessionWithEnrichedData[]> {
    const res = await api.get<{ items: SessionWithEnrichedData[] }>(
        `/api/v1/attendance/sessions/class/${classId}/date/${date}`
    );
    return res.items ?? res;
}

/** Update an attendance session. */
export async function updateSession(
    id: string,
    data: UpdateSessionPayload
): Promise<SessionWithEnrichedData> {
    return api.put<SessionWithEnrichedData>(`/api/v1/attendance/sessions/${id}`, data);
}

// ─── Records ──────────────────────────────────────────────────────────────

/** Batch mark attendance for multiple students in a single slot+date. */
export async function batchMarkAttendance(
    data: BatchMarkPayload,
    termId?: string
): Promise<BatchMarkResult> {
    const qs = termId ? `?term_id=${encodeURIComponent(termId)}` : "";
    return api.post<BatchMarkResult>(`/api/v1/attendance/records/batch${qs}`, data);
}

/** List attendance records by timetable slot and date. */
export async function listRecordsBySlot(
    timetableSlotId: string,
    date: string
): Promise<RecordWithEnrichedData[]> {
    const res = await api.get<{ items: RecordWithEnrichedData[] }>(
        `/api/v1/attendance/records/slot?timetable_slot_id=${encodeURIComponent(timetableSlotId)}&date=${encodeURIComponent(date)}`
    );
    return res.items ?? res;
}

/** List attendance records for a student in a term. */
export async function listRecordsByStudent(
    studentId: string,
    termId?: string
): Promise<RecordWithEnrichedData[]> {
    const qs = termId ? `?term_id=${encodeURIComponent(termId)}` : "";
    const res = await api.get<{ items: RecordWithEnrichedData[] }>(
        `/api/v1/attendance/records/student/${studentId}${qs}`
    );
    return res.items ?? res;
}

/** List attendance records for a class on a date. */
export async function listRecordsByClassDate(
    classId: string,
    date: string,
    termId?: string
): Promise<RecordWithEnrichedData[]> {
    let path = `/api/v1/attendance/records/class/${classId}/date/${date}`;
    if (termId) path += `?term_id=${encodeURIComponent(termId)}`;
    const res = await api.get<{ items: RecordWithEnrichedData[] }>(path);
    return res.items ?? res;
}

/** List attendance records with filters. */
export async function listRecords(
    params: {
        timetable_slot_id?: string;
        date?: string;
        student_id?: string;
        class_id?: string;
        academic_term_id?: string;
        status?: string;
    } = {}
): Promise<RecordListResponse> {
    const searchParams = new URLSearchParams();
    if (params.timetable_slot_id) searchParams.set("timetable_slot_id", params.timetable_slot_id);
    if (params.date) searchParams.set("date", params.date);
    if (params.student_id) searchParams.set("student_id", params.student_id);
    if (params.class_id) searchParams.set("class_id", params.class_id);
    if (params.academic_term_id) searchParams.set("academic_term_id", params.academic_term_id);
    if (params.status) searchParams.set("status", params.status);
    const qs = searchParams.toString();
    return api.get<RecordListResponse>(`/api/v1/attendance/records?${qs}`);
}

/** Update a single attendance record. */
export async function updateRecord(
    id: string,
    data: UpdateRecordPayload
): Promise<RecordWithEnrichedData> {
    return api.put<RecordWithEnrichedData>(`/api/v1/attendance/records/${id}`, data);
}

// ─── Summaries ────────────────────────────────────────────────────────────

/** Get attendance summaries for a student in a term. */
export async function getStudentTermSummary(
    studentId: string,
    termId?: string
): Promise<AttendanceTermSummary[]> {
    const qs = termId ? `?term_id=${encodeURIComponent(termId)}` : "";
    const res = await api.get<{ items: AttendanceTermSummary[] }>(
        `/api/v1/attendance/summaries/student/${studentId}${qs}`
    );
    return res.items ?? res;
}

/** Get attendance summaries for all students in a class for a term. */
export async function getClassTermSummary(
    classId: string,
    termId?: string
): Promise<AttendanceTermSummary[]> {
    const qs = termId ? `?term_id=${encodeURIComponent(termId)}` : "";
    const res = await api.get<{ items: AttendanceTermSummary[] }>(
        `/api/v1/attendance/summaries/class/${classId}${qs}`
    );
    return res.items ?? res;
}

/** Refresh materialised attendance summaries for a term. */
export async function refreshSummaries(termId: string): Promise<RefreshSummaryResponse> {
    return api.post<RefreshSummaryResponse>("/api/v1/attendance/summaries/refresh", {
        term_id: termId,
    });
}

// ─── Calendar Status ───────────────────────────────────────────────────────

/**
 * Get per-date attendance completion status for a school calendar month view.
 * Returns one entry per date in the range with expected/handled counts and a computed status.
 */
export async function getCalendarStatus(
    startDate: string,
    endDate: string
): Promise<CalendarStatusListResponse> {
    const qs = new URLSearchParams({ start_date: startDate, end_date: endDate }).toString();
    return api.get<CalendarStatusListResponse>(`/api/v1/attendance/calendar/status?${qs}`);
}
