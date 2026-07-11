/**
 * Academic Terms API functions.
 *
 * Endpoints:
 *   GET /api/v1/academic-terms — list terms (optionally by academic_year_id)
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

// ─── API Functions ─────────────────────────────────────────────────────────

/** List academic terms for the active school. */
export async function listTerms(
    params: { academic_year_id?: string } = {}
): Promise<{ items: AcademicTerm[] }> {
    const searchParams = new URLSearchParams();
    if (params.academic_year_id) searchParams.set("academic_year_id", params.academic_year_id);

    const qs = searchParams.toString();
    return api.get<{ items: AcademicTerm[] }>(`/api/v1/academic-terms?${qs}`);
}

/** List academic years for the active school.
 *
 * The backend returns { data: AcademicYearWithTerms[] }. This normalizes it
 * to the { items: AcademicYear[] } shape expected by the frontend. */
export async function listAcademicYears(): Promise<{ items: AcademicYear[] }> {
    const raw = await api.get<{ data: AcademicYear[] }>("/api/v1/academic-years");
    return { items: raw.data ?? [] };
}
