/**
 * Schools API functions.
 *
 * Endpoints:
 *   GET  /schools?tenant_id=...  — list schools under a tenant
 *   POST /schools                — create a new school
 *   PUT  /schools/:id            — update a school name
 *   DELETE /schools/:id          — soft-delete a school
 *
 * 🔄 AUTO-GENERATED TYPES: See src/lib/api/generated.ts (generated from backend swagger.json).
 */

import { api } from "./client";
import type { definitions } from "./generated";

export type School = definitions["internal_school.School"];
export type CreateSchoolPayload = definitions["internal_school.CreateSchoolPayload"];

export interface UpdateSchoolPayload {
    name: string;
}

/** List all active schools for a tenant. */
export async function listSchools(tenantId: string): Promise<School[]> {
    return api.get<School[]>(`/schools?tenant_id=${encodeURIComponent(tenantId)}`);
}

/** Create a new school (requires SCHOOL_ADMIN role). */
export async function createSchool(payload: CreateSchoolPayload): Promise<School> {
    return api.post<School>("/schools", payload);
}

/** Update a school's name (requires SCHOOL_ADMIN or SYSTEM_ADMIN role). */
export async function updateSchool(
    schoolId: string,
    payload: UpdateSchoolPayload
): Promise<School> {
    return api.put<School>(`/schools/${encodeURIComponent(schoolId)}`, payload);
}

/** Soft-delete a school (requires SCHOOL_ADMIN or SYSTEM_ADMIN role). */
export async function deleteSchool(schoolId: string): Promise<void> {
    return api.delete<void>(`/schools/${encodeURIComponent(schoolId)}`);
}

/** Activate a school — switch the user's current active school membership. */
export async function activateSchool(schoolId: string): Promise<School> {
    return api.post<School>(`/schools/${encodeURIComponent(schoolId)}/activate`);
}
