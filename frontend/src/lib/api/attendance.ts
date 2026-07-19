/**
 * Attendance API functions.
 *
 * Endpoints:
 *   GET    /api/v1/attendance/roster/:timetable_slot_id  — teacher roster for a slot
 *   POST   /api/v1/attendance/bulk                        — bulk mark attendance
 *   GET    /api/v1/attendance/dashboard                   — admin completion dashboard
 *   GET    /api/v1/attendance/students/:student_id        — student history
 *   PUT    /api/v1/attendance/records/:id                  — admin single-record correction
 *   GET    /api/v1/attendance/children/:student_id/summary — parent-facing summary
 *   POST   /api/v1/attendance/summaries/compute           — trigger summary recompute
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export type AttendanceStatus = "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";

export interface RosterStudent {
    student_id: string;
    full_name: string;
    admission_number: string;
    current_status?: AttendanceStatus | null;
}

export interface SlotRosterResponse {
    timetable_slot_id: string;
    class_name: string;
    learning_area: string;
    date: string;
    students: RosterStudent[];
}

export interface BulkAttendanceEntry {
    student_id: string;
    status: AttendanceStatus;
    note?: string | null;
}

export interface BulkAttendancePayload {
    timetable_slot_id: string;
    date: string;
    entries: BulkAttendanceEntry[];
}

export interface StudentAttendanceRecord {
    date: string;
    subject: string;
    status: AttendanceStatus;
}

export interface ChildAttendanceSummary {
    student_id: string;
    term_id: string;
    attendance_percentage: number;
    recent_periods: StudentAttendanceRecord[];
}

export interface CompletionStatus {
    class_id: string;
    class_name: string;
    slot_id: string;
    period_name: string;
    learning_area: string;
    learning_area_id: string;
    total_slots: number;
    marked_slots: number;
    is_complete: boolean;
}

export interface AdminDashboardResponse {
    date: string;
    items: CompletionStatus[];
    total: number;
    page: number;
    limit: number;
}

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
    marked_at: string;
    note?: string | null;
    created_at: string;
    updated_at: string;
}

// ─── API Functions ────────────────────────────────────────────────────────

/** Get the roster for a timetable slot with existing marks. */
export async function getSlotRoster(
    timetableSlotId: string,
    date?: string
): Promise<SlotRosterResponse> {
    const params = new URLSearchParams();
    if (date) params.set("date", date);
    const qs = params.toString();
    const path = `/api/v1/attendance/roster/${timetableSlotId}${qs ? `?${qs}` : ""}`;
    return api.get<SlotRosterResponse>(path);
}

/** Bulk mark attendance for a slot/date. */
export async function bulkMarkAttendance(
    payload: BulkAttendancePayload
): Promise<{ message: string; count: number }> {
    return api.post<{ message: string; count: number }>("/api/v1/attendance/bulk", payload);
}

/** Params for listing admin attendance dashboard items with pagination and filters. */
export interface ListAdminAttendancesParams {
    date?: string;
    search?: string;
    page?: number;
    limit?: number;
    /** Filter values keyed by FilterItem id, e.g. { education_level: ["Early_Years"], grade_level: ["G4"], class_id: "uuid", is_complete: "complete" } */
    filters?: Record<string, string | string[]>;
}

/** List admin attendance dashboard items with pagination and filters. */
export async function listAdminAttendances(
    params: ListAdminAttendancesParams = {}
): Promise<AdminDashboardResponse> {
    const searchParams = new URLSearchParams();

    if (params.date) searchParams.set("date", params.date);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    // Multi-value filters from DataTable filter groups
    const filters = params.filters ?? {};

    const edLevels = asFilterArray(filters, "education_level");
    for (const el of edLevels) {
        searchParams.append("education_level", el);
    }

    const grLevels = asFilterArray(filters, "grade_level");
    for (const gl of grLevels) {
        searchParams.append("grade_level", gl);
    }

    const classId = filters["class_id"];
    if (typeof classId === "string" && classId) {
        searchParams.set("class_id", classId);
    }

    const isComplete = filters["is_complete"];
    if (typeof isComplete === "string" && isComplete) {
        searchParams.set("is_complete", isComplete);
    }

    const qs = searchParams.toString();
    const path = qs ? `/api/v1/attendance/dashboard?${qs}` : "/api/v1/attendance/dashboard";
    return api.get<AdminDashboardResponse>(path);
}

/** Safely coerces a filter value to a string array. */
function asFilterArray(filters: Record<string, string | string[]>, key: string): string[] {
    const val = filters[key];
    if (Array.isArray(val)) return val;
    if (typeof val === "string" && val) return [val];
    return [];
}

/** Get admin dashboard — completion status per class for a date (legacy). */
export async function getAdminDashboard(date?: string): Promise<AdminDashboardResponse> {
    const params = new URLSearchParams();
    if (date) params.set("date", date);
    const qs = params.toString();
    return api.get<AdminDashboardResponse>(`/api/v1/attendance/dashboard${qs ? `?${qs}` : ""}`);
}

/** Get attendance history for a specific student. */
export async function getStudentHistory(
    studentId: string,
    filters?: { term_id?: string; start_date?: string; end_date?: string }
): Promise<{ items: AttendanceRecord[]; total: number }> {
    const params = new URLSearchParams();
    if (filters?.term_id) params.set("term_id", filters.term_id);
    if (filters?.start_date) params.set("start_date", filters.start_date);
    if (filters?.end_date) params.set("end_date", filters.end_date);
    const qs = params.toString();
    return api.get<{ items: AttendanceRecord[]; total: number }>(
        `/api/v1/attendance/students/${studentId}${qs ? `?${qs}` : ""}`
    );
}

/** Update a single attendance record (admin correction). */
export async function updateAttendanceRecord(
    recordId: string,
    payload: { status: AttendanceStatus; note?: string | null }
): Promise<{ message: string }> {
    return api.put<{ message: string }>(`/api/v1/attendance/records/${recordId}`, payload);
}

/** Get a parent-facing attendance summary for a child. */
export async function getChildAttendanceSummary(
    studentId: string,
    termId: string
): Promise<ChildAttendanceSummary> {
    return api.get<ChildAttendanceSummary>(
        `/api/v1/attendance/children/${studentId}/summary?term_id=${termId}`
    );
}

// ─── Attendance Session Types & API ──────────────────────────────────────

export type SessionStatus = "SUBMITTED" | "SKIPPED";

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

export interface SkipSessionPayload {
    timetable_slot_id: string;
    date: string;
    skip_reason: string;
}

/** Mark a session as skipped (class did not hold). */
export async function skipSession(payload: SkipSessionPayload): Promise<{ message: string }> {
    return api.post<{ message: string }>("/api/v1/attendance/sessions/skip", payload);
}

/** Unskip a session (re-open so attendance can be marked again). */
export async function unskipSession(payload: {
    timetable_slot_id: string;
    date: string;
}): Promise<{ message: string }> {
    return api.post<{ message: string }>("/api/v1/attendance/sessions/unskip", payload);
}

/** Get session status for a slot + date. Returns { session: AttendanceSession | null }. */
export async function getSession(
    timetableSlotId: string,
    date: string
): Promise<{ session: AttendanceSession | null }> {
    const params = new URLSearchParams({ date });
    return api.get<{ session: AttendanceSession | null }>(
        `/api/v1/attendance/sessions/${timetableSlotId}?${params.toString()}`
    );
}

/** Trigger recomputation of attendance term summaries. */
export async function computeAttendanceSummaries(
    termId: string
): Promise<{ message: string; count: number }> {
    return api.post<{ message: string; count: number }>("/api/v1/attendance/summaries/compute", {
        term_id: termId,
    });
}
