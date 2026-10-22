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

export interface ListAcademicPeriodResponse {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    terms: TermResponse[];
}

export interface TermResponse {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** Create an academic period (year + terms) for the active school. */
export async function createAcademicPeriod(
    payload: AcademicPeriodRequest
): Promise<CreateAcademicPeriodResponse> {
    return api.post<CreateAcademicPeriodResponse>("/api/school/academic-period", payload);
}

export interface ListAcademicPeriodsResponse {
    code: string;
    message: string;
    data: ListAcademicPeriodResponse[];
    errors: Record<string, string[]>;
}

/** List all academic periods (years) with their terms for the active school. */
export async function listAcademicPeriods(): Promise<ListAcademicPeriodsResponse> {
    return api.get<ListAcademicPeriodsResponse>("/api/school/academic-period");
}
