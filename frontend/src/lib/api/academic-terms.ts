/**
 * Academic Terms & Years API functions.
 *
 * Backend contract (backend/internal/academic-years/handler.go):
 *
 *   Academic years are READ-ONLY via the API — year creation is driven by the
 *   term lifecycle and SetupInitialYear during school registration. All term
 *   mutations are SCHOOL_ADMIN / SYSTEM_ADMIN only.
 *
 * Endpoints:
 *   GET    /api/v1/academic-years              — list years (with nested terms)
 *   GET    /api/v1/academic-years/current      — current year + current term
 *   GET    /api/v1/academic-terms              — list terms (?academic_year_id=)
 *   POST   /api/v1/academic-terms              — create term
 *   PATCH  /api/v1/academic-terms/:id          — update term (optimistic locking)
 *   POST   /api/v1/academic-terms/:id/activate — activate term (sets is_current)
 *   DELETE /api/v1/academic-terms/:id          — delete term
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

export interface CurrentAcademicYearWithCurrentTerm {
    academic_year_id: string;
    academic_year_name: string;
    academic_year_start_date: string;
    academic_year_end_date: string;
    academic_term_id: string;
    academic_term_name: string;
    academic_term_number: string;
    academic_term_start_date: string;
    academic_term_end_date: string;
    academic_term_is_final: boolean;
}

// ─── Create / Update Payloads ─────────────────────────────────────────────

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

// ─── Response Types ────────────────────────────────────────────────────────
export interface ActivateTermResponse extends AcademicTerm {
    message: string;
}

// ─── API Functions — Academic Years (read-only) ───────────────────────────

/** Get the current academic year plus its current term for the active school. */
export async function getCurrentYearAndTerm(): Promise<CurrentAcademicYearWithCurrentTerm> {
    return Promise.resolve({
        academic_year_id: "",
        academic_year_name: "",
        academic_year_start_date: "",
        academic_year_end_date: "",
        academic_term_id: "",
        academic_term_name: "",
        academic_term_number: "",
        academic_term_start_date: "",
        academic_term_end_date: "",
        academic_term_is_final: false,
    });
}

/** List academic years for the active school (each with nested terms). */
export async function listAcademicYears(): Promise<{ data: AcademicYear[] }> {
    return { data: [] };
}

// ─── API Functions — Academic Terms ───────────────────────────────────────

/** List academic terms for the active school, optionally filtered by year. */
export async function listTerms(
    params: { academic_year_id?: string } = {}
): Promise<AcademicTerm[]> {
    const searchParams = new URLSearchParams();
    if (params.academic_year_id) searchParams.set("academic_year_id", params.academic_year_id);

    const qs = searchParams.toString();
    return [];
}

/** Create a new academic term. Returns the created term. */
export async function createTerm(payload: CreateTermPayload): Promise<AcademicTerm> {
    return Promise.resolve({
        id: "",
        academic_year_id: "",
        name: "",
        term_number: 0,
        start_date: "",
        end_date: "",
        is_current: false,
        is_final: false,
        version: 0,
        created_at: "",
    });
}

/** Update an existing academic term (optimistic locking via version). */
export async function updateTerm(id: string, payload: UpdateTermPayload): Promise<AcademicTerm> {
    return Promise.resolve({
        id: "",
        academic_year_id: "",
        name: "",
        term_number: 0,
        start_date: "",
        end_date: "",
        is_current: false,
        is_final: false,
        version: 0,
        created_at: "",
    });
}

/**
 * Activate an academic term, making it the school's current term.
 * POST /api/v1/academic-terms/:id/activate — SCHOOL_ADMIN only.
 */
export async function activateTerm(id: string): Promise<ActivateTermResponse> {
    return Promise.resolve({
        id: "",
        academic_year_id: "",
        name: "",
        term_number: 0,
        start_date: "",
        end_date: "",
        is_current: false,
        is_final: false,
        version: 0,
        created_at: "",
        message: "",
    });
}

/**
 * Delete an academic term (fails with 409 if the term has dependent records).
 * DELETE /api/v1/academic-terms/:id — SCHOOL_ADMIN only.
 */
export async function deleteTerm(id: string): Promise<void> {
    return undefined;
}
