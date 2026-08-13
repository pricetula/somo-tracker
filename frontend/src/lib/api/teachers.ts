/**
 * Teachers API functions.
 *
 * Endpoints:
 *   GET  /api/v1/teachers
 *   PATCH /api/v1/teachers/:user_id/active
 *   DELETE /api/v1/teachers/:user_id
 */

import { api } from "./client";
import type { TeacherMember, ListTeachersResponse } from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { TeacherMember, ListTeachersResponse };

// ─── API Functions ─────────────────────────────────────────────────────────

/** List teachers with extended fields (TSC, KNEC, teacher_role). */
export async function listTeachers(
    params: {
        page?: number;
        limit?: number;
        search?: string;
        include_inactive?: boolean;
        /** Filter values keyed by FilterItem id, e.g. { education_level: ["Early_Years"], grade_level: ["G1", "G2"] } */
        filters?: Record<string, string[]>;
    } = {}
): Promise<ListTeachersResponse> {
    const searchParams = new URLSearchParams();

    // Pass multi-value filter params
    const edLevels = params.filters?.education_level ?? [];
    for (const el of edLevels) {
        searchParams.append("education_level", el);
    }
    const grLevels = params.filters?.grade_level ?? [];
    for (const gl of grLevels) {
        searchParams.append("grade_level", gl);
    }

    if (params.search) searchParams.set("search", params.search);
    if (params.include_inactive) searchParams.set("include_inactive", "true");
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ListTeachersResponse>(`/api/v1/teachers?${qs}`);
}

/** Get a single teacher by ID. */
export async function getTeacher(userId: string): Promise<TeacherMember> {
    return api.get<TeacherMember>(`/api/v1/teachers/${userId}`);
}

/** Update a teacher's profile (TSC number, KNEC panel assessor, name). */
export async function updateTeacher(
    userId: string,
    payload: {
        full_name?: string;
        tsc_number?: string | null;
        knec_panel_assessor_id?: string | null;
    }
): Promise<void> {
    return api.put<void>(`/api/v1/teachers/${userId}`, payload);
}

/** Toggle teacher active status. */
export async function toggleTeacherActive(userId: string, isActive: boolean): Promise<void> {
    return api.patch<void>(`/api/v1/teachers/${userId}/active`, { is_active: isActive });
}

/** Hard-delete a teacher. */
export async function deleteTeacher(userId: string): Promise<void> {
    return api.delete<void>(`/api/v1/teachers`, { user_id: userId });
}

/**
 * List classes assigned to a teacher, optionally for a specific term.
 * GET /api/v1/teachers/:user_id/classes?term_id=
 */
export async function listTeacherClasses(
    userId: string,
    termId?: string
): Promise<{ items: unknown[]; total: number }> {
    const qs = termId ? `?term_id=${encodeURIComponent(termId)}` : "";
    return api.get<{ items: unknown[]; total: number }>(`/api/v1/teachers/${userId}/classes${qs}`);
}

/**
 * Get a teacher's timetable for a day of the week (1=Monday … 7=Sunday).
 * GET /api/v1/teachers/:user_id/timetable?day_of_week=
 */
export async function getTeacherTimetable(
    userId: string,
    dayOfWeek = 1
): Promise<{ items: unknown[]; total: number }> {
    return api.get<{ items: unknown[]; total: number }>(
        `/api/v1/teachers/${userId}/timetable?day_of_week=${dayOfWeek}`
    );
}
