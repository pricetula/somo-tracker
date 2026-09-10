/**
 * Schools API — new school registration only.
 * Previous endpoints (list/update/delete/seed) are deprecated.
 *
 * Endpoint (from backend/internal/api/school_handler.go, swaggo annotated):
 *   POST /api/school/register — register a new school and assign ADMIN
 */

import { api } from "./client";

export interface RegisterSchoolPayload {
    school_name: string;
    user_name: string;
}

export interface RegisterSchoolResponse {
    code: string;
    message: string;
    school_id: string;
    school_name: string;
    user_name: string;
    errors: Record<string, unknown>;
}

/** Register a new school (atomic: updates user + creates school + creates ADMIN membership). */
export async function registerSchool(data: RegisterSchoolPayload): Promise<RegisterSchoolResponse> {
    return api.post<RegisterSchoolResponse>("/api/school/register", data);
}
