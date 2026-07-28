/**
 * Schools API functions.
 *
 * Endpoints (from backend/internal/cbcschools/handler.go):
 *   POST   /api/v1/schools      — create a school
 *   GET    /api/v1/schools       — list all schools for the tenant
 *   PUT    /api/v1/schools/:id   — update a school
 *   DELETE /api/v1/schools/:id   — delete a school
 */

import { api } from "./client";
import type {
    SchoolWithMemberCount,
    ListSchoolsResponse,
    CreateSchoolPayload,
    CreateSchoolResponse,
} from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type {
    SchoolWithMemberCount,
    ListSchoolsResponse,
    CreateSchoolPayload,
    CreateSchoolResponse,
};

// ─── API Functions ─────────────────────────────────────────────────────────

/** List all schools for the current user's tenant. */
export async function listSchools(): Promise<ListSchoolsResponse> {
    return api.get<ListSchoolsResponse>("/api/v1/schools");
}

/** Create a new school. */
export async function createSchool(data: CreateSchoolPayload): Promise<CreateSchoolResponse> {
    return api.post<CreateSchoolResponse>("/api/v1/schools", data);
}

/** Update a school's details. */
export async function updateSchool(
    id: string,
    payload: { name?: string; code?: string }
): Promise<void> {
    return api.put<void>(`/api/v1/schools/${id}`, payload);
}

/** Delete a school. */
export async function deleteSchool(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/schools`, { id });
}

/** Set a school as the active school for the current user. */
export async function setActiveSchool(schoolId: string): Promise<void> {
    return api.post<void>(`/api/v1/schools/${schoolId}/activate`);
}

/** Seed a school with learning areas. */
export async function seedSchool(): Promise<void> {
    return api.post<void>("/api/v1/schools/seed-curriculum");
}
