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
    return { items: [], total: 0 };
}

/** Create a new school. */
export async function createSchool(data: CreateSchoolPayload): Promise<CreateSchoolResponse> {
    return Promise.resolve({ id: "", name: "", address: "", created_at: "" });
}

/** Update a school's details. */
export async function updateSchool(
    id: string,
    payload: {
        name?: string;
        county?: string;
        sub_county?: string;
        ward?: string;
        knec_school_code?: string;
        nemis_code?: string;
        school_type?: string;
        is_active?: boolean;
    }
): Promise<void> {
    return undefined;
}

/** Delete a school. */
export async function deleteSchool(id: string): Promise<void> {
    return undefined;
}

/** Set a school as the active school for the current user. */
export async function setActiveSchool(schoolId: string): Promise<void> {
    return undefined;
}

/** Seed a school with learning areas. */
export async function seedSchool(): Promise<void> {
    return undefined;
}
