/**
 * Teachers API functions.
 *
 * Endpoints:
 *   GET  /api/v1/teachers
 *   PATCH /api/v1/teachers/:user_id/active
 */

import { api } from "./client";
import type { TeacherMember, ListTeachersResponse } from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { TeacherMember, ListTeachersResponse };

// ─── API Functions ─────────────────────────────────────────────────────────

/** List teachers with extended fields (TSC, KNEC, teacher_role). */
export async function listTeachers(
    params: { page?: number; per_page?: number; search?: string; include_inactive?: boolean } = {}
): Promise<ListTeachersResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.per_page) searchParams.set("per_page", String(params.per_page));
    if (params.search) searchParams.set("search", params.search);
    if (params.include_inactive) searchParams.set("include_inactive", "true");

    const qs = searchParams.toString();
    return api.get<ListTeachersResponse>(`/api/v1/teachers?${qs}`);
}

/** Toggle teacher active status. */
export async function toggleTeacherActive(userId: string, isActive: boolean): Promise<void> {
    return api.patch<void>(`/api/v1/teachers/${userId}/active`, { is_active: isActive });
}
