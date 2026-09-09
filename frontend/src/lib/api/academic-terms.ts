/**
 * Academic Period API functions.
 *
 * Backend contract (backend/internal/api/academic_period_handler.go):
 *
 *   POST   /school/academic-period    — create academic year + nested terms
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface TermInput {
    name: string;
    start_date: string; // "YYYY-MM-DD"
    end_date: string; // "YYYY-MM-DD"
}

export interface AcademicPeriodRequest {
    year: number;
    terms: TermInput[];
}

export interface CreateAcademicPeriodResponse {
    code: string;
    message: string;
    academic_year_id: string;
    errors: Record<string, string[]>;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** Create an academic period (year + terms) for the active school. */
export async function createAcademicPeriod(
    payload: AcademicPeriodRequest
): Promise<CreateAcademicPeriodResponse> {
    return api.post<CreateAcademicPeriodResponse>("/school/academic-period", payload);
}
