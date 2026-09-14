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

export interface CreateSchoolPayload {
    school_name: string;
}

export interface CreateSchoolResponse {
    code: string;
    message: string;
    school_id: string;
    school_name: string;
    errors: Record<string, unknown>;
}

/** Create a new school (admin-only, sets up academic periods and CBE curriculum). */
export async function createSchool(data: CreateSchoolPayload): Promise<CreateSchoolResponse> {
    return api.post<CreateSchoolResponse>("/api/school", data);
}

export interface SchoolItem {
    id: string;
    name: string;
    country_name?: string;
    education_system_name?: string;
    /** Legacy fields kept for compatibility */
    role?: string;
    is_active?: boolean;
}

export interface ListSchoolsResponse {
    code: string;
    message: string;
    schools: SchoolItem[];
    errors: Record<string, unknown>;
}

/** List schools for the authenticated user in current tenant. */
export async function listSchools(): Promise<ListSchoolsResponse> {
    return api.get<ListSchoolsResponse>("/api/schools");
}
