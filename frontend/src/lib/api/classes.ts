/**
 * Classes API functions.
 *
 * Endpoints:
 *   GET  /api/v1/classes — list classes (filters optional, backend defaults to current academic year)
 *   POST /api/v1/classes — create class
 *   PUT  /api/v1/classes/:id — update class
 *   DELETE /api/v1/classes — bulk delete classes
 */

import { api } from "./client";
import type { Class, ClassListResult } from "./generated";

export type { Class, ClassListResult };

/** List classes params, supporting filters from DataTable filter groups. */
export interface ListClassesParams {
    academic_year_id?: string;
    academic_term_id?: string;
    grade_level?: string;
    stream_id?: string;
    search?: string;
    page?: number;
    limit?: number;
    /** Filter values keyed by FilterItem id, e.g. { grade_level: ["G1", "G2"], stream_id: ["id1"], academic_year_id: ["ay1"] } */
    filters?: Record<string, string[]>;
}

/** List classes for the active school. */
export async function listClasses(params: ListClassesParams = {}): Promise<ClassListResult> {
    const searchParams = new URLSearchParams();

    // Multi-value filters from DataTable
    const gradeLevels = params.filters?.grade_level ?? [];
    for (const gl of gradeLevels) {
        searchParams.append("grade_level", gl);
    }
    const streamIds = params.filters?.stream_id ?? [];
    for (const sid of streamIds) {
        searchParams.append("stream_id", sid);
    }
    const academicYearIds = params.filters?.academic_year_id ?? [];
    for (const ayid of academicYearIds) {
        searchParams.append("academic_year_id", ayid);
    }

    if (params.academic_year_id) searchParams.set("academic_year_id", params.academic_year_id);
    if (params.academic_term_id) searchParams.set("academic_term_id", params.academic_term_id);
    if (params.grade_level) searchParams.set("grade_level", params.grade_level);
    if (params.stream_id) searchParams.set("stream_id", params.stream_id);
    if (params.search) searchParams.set("search", params.search);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ClassListResult>(`/api/v1/classes?${qs}`);
}

/**
 * Get a single class by ID.
 * GET /api/v1/classes/:id
 */
export async function getClass(classId: string): Promise<Class> {
    return api.get<Class>(`/api/v1/classes/${classId}`);
}

/** Bulk delete classes. */
export async function bulkDeleteClasses(ids: string[]): Promise<void> {
    await api.delete("/api/v1/classes", { class_ids: ids });
}

/**
 * Create a new class.
 * POST /api/v1/classes
 */
export async function createClass(payload: {
    grade_level: string;
    stream_id: string;
    student_ids?: string[];
}): Promise<Class> {
    return api.post<Class>("/api/v1/classes", payload);
}

// ─── Enrollment Types ───────────────────────────────────────────────────────

/** A single student enrolled in a class roster. */
export interface RosterEntry {
    id: string;
    full_name: string;
    admission_number?: string;
    upi_number?: string;
    gender: string;
    enrolled_at?: string;
}

/** Paginated roster list result. */
export interface RosterListResult {
    items: RosterEntry[];
    total: number;
    page: number;
    limit: number;
}

/** A student available for enrollment (not in this class). */
export interface AvailableStudent {
    id: string;
    full_name: string;
    admission_number?: string | null;
    upi_number?: string | null;
    gender: string;
    current_class?: string | null;
    current_class_id?: string | null;
}

/** Response for available students listing. */
export interface AvailableStudentsResponse {
    items: AvailableStudent[];
    total: number;
    page: number;
    limit: number;
}

/** Response for batch enrollment. */
export interface BatchEnrollResponse {
    code: string;
    message: string;
    enrolled_count: number;
}

// ─── Enrollment API Functions ───────────────────────────────────────────────

/**
 * Get the roster of students enrolled in a class for the current term.
 * GET /api/v1/classes/:id/roster
 */
export async function getClassRoster(
    classId: string,
    params: {
        page?: number;
        limit?: number;
        search?: string;
    } = {}
): Promise<RosterListResult> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    const qs = searchParams.toString();
    return api.get<RosterListResult>(`/api/v1/classes/${classId}/roster?${qs}`);
}

/**
 * Batch enroll students into a class.
 * POST /api/v1/classes/:id/enroll
 */
export async function batchEnrollStudents(
    classId: string,
    studentIds: string[]
): Promise<BatchEnrollResponse> {
    return api.post<BatchEnrollResponse>(`/api/v1/classes/${classId}/enroll`, {
        student_ids: studentIds,
    });
}

/**
 * Unenroll a single student from a class for a specific term.
 * POST /api/v1/classes/:id/unenroll/:studentId?academic_term_id=...
 */
export async function unenrollStudent(classId: string, studentId: string): Promise<void> {
    return api.post<void>(`/api/v1/classes/${classId}/unenroll/${studentId}`);
}

/**
 * Get students available for enrollment (not in this class).
 * GET /api/v1/classes/:id/available-students
 */
export async function getAvailableStudents(
    classId: string,
    params: {
        search?: string;
        page?: number;
        limit?: number;
    } = {}
): Promise<AvailableStudentsResponse> {
    const searchParams = new URLSearchParams();
    if (params.search) searchParams.set("search", params.search);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    const qs = searchParams.toString();
    return api.get<AvailableStudentsResponse>(
        `/api/v1/classes/${classId}/available-students?${qs}`
    );
}
