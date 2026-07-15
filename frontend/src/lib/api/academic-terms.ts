/**
 * Academic Terms & Years API functions.
 *
 * Endpoints:
 *   GET    /api/v1/academic-years         — list years
 *   POST   /api/v1/academic-years         — create year
 *   PATCH  /api/v1/academic-years/:id     — update year
 *   POST   /api/v1/academic-years/:id/set-current — set current year
 *   DELETE /api/v1/academic-years/:id     — delete year
 *   GET    /api/v1/academic-terms         — list terms
 *   POST   /api/v1/academic-terms         — create term
 *   PATCH  /api/v1/academic-terms/:id     — update term
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface AcademicTerm {
    id: string;
    academic_year_id: string;
    name: string;
    term_number: number;
    start_date: string;
    end_date: string;
    is_current: boolean;
    is_final: boolean;
    version: number;
    created_at: string;
}

export interface AcademicYear {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    is_current: boolean;
    version: number;
    created_at: string;
    terms?: AcademicTerm[];
}

// ─── Create / Update Payloads ─────────────────────────────────────────────

export interface CreateAcademicYearPayload {
    name: string;
    start_date: string; // "YYYY-MM-DD"
    end_date: string; // "YYYY-MM-DD"
}

export interface UpdateAcademicYearPayload {
    name?: string;
    start_date?: string; // "YYYY-MM-DD"
    end_date?: string; // "YYYY-MM-DD"
    version: number; // required for optimistic locking
}

export interface CreateTermPayload {
    academic_year_id: string;
    name: string;
    term_number: number;
    start_date: string; // "YYYY-MM-DD"
    end_date: string; // "YYYY-MM-DD"
}

export interface UpdateTermPayload {
    name?: string;
    start_date?: string; // "YYYY-MM-DD"
    end_date?: string; // "YYYY-MM-DD"
    version: number; // required for optimistic locking
}

// ─── API Functions — Academic Years ────────────────────────────────────────

/** List academic years for the active school. */
export async function listAcademicYears(): Promise<{ items: AcademicYear[] }> {
    const raw = await api.get<{ data: AcademicYear[] }>("/api/v1/academic-years");
    return { items: raw.data ?? [] };
}

/** Create a new academic year. Returns the new year's ID. */
export async function createAcademicYear(
    payload: CreateAcademicYearPayload
): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/academic-years", payload);
}

/** Update an existing academic year (optimistic locking via version). */
export async function updateAcademicYear(
    id: string,
    payload: UpdateAcademicYearPayload
): Promise<{ id: string; version: number }> {
    return api.patch<{ id: string; version: number }>(`/api/v1/academic-years/${id}`, payload);
}

/** Set an academic year as the current year for the school. */
export async function setCurrentYear(id: string): Promise<void> {
    await api.post(`/api/v1/academic-years/${id}/set-current`);
}

/** Delete an academic year and its cascade-deleted terms. */
export async function deleteAcademicYear(id: string): Promise<void> {
    await api.delete(`/api/v1/academic-years/${id}`);
}

// ─── API Functions — Academic Terms ───────────────────────────────────────

/** List academic terms for the active school, optionally filtered by year. */
export async function listTerms(
    params: { academic_year_id?: string } = {}
): Promise<{ items: AcademicTerm[] }> {
    const searchParams = new URLSearchParams();
    if (params.academic_year_id) searchParams.set("academic_year_id", params.academic_year_id);

    const qs = searchParams.toString();
    const raw = await api.get<{ data: AcademicTerm[] }>(`/api/v1/academic-terms?${qs}`);
    return { items: raw.data ?? [] };
}

/** Create a new academic term. Returns the created term. */
export async function createTerm(payload: CreateTermPayload): Promise<AcademicTerm> {
    return api.post<AcademicTerm>("/api/v1/academic-terms", payload);
}

/** Update an existing academic term (optimistic locking via version). */
export async function updateTerm(id: string, payload: UpdateTermPayload): Promise<AcademicTerm> {
    return api.patch<AcademicTerm>(`/api/v1/academic-terms/${id}`, payload);
}
