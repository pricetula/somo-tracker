/**
 * Classes API functions.
 *
 * Endpoints:
 *   GET  /api/v1/classes — list classes (filters optional, backend defaults to current academic year)
 *   POST /api/v1/classes — create class
 *   PUT  /api/v1/classes/:id — update class
 *   DELETE /api/v1/classes — bulk delete classes
 */

import { api } from "./client";
import type { Class, ClassListResult } from "./generated";

export type { Class, ClassListResult };

/** List classes params, supporting filters from DataTable filter groups. */
export interface ListClassesParams {
    academic_year_id?: string;
    academic_term_id?: string;
    grade_level?: string;
    stream_id?: string;
    search?: string;
    page?: number;
    limit?: number;
    /** Filter values keyed by FilterItem id, e.g. { grade_level: ["G1", "G2"], stream_id: ["id1"], academic_year_id: ["ay1"] } */
    filters?: Record<string, string[]>;
}

/** List classes for the active school. */
export async function listClasses(params: ListClassesParams = {}): Promise<ClassListResult> {
    const searchParams = new URLSearchParams();

    // Multi-value filters from DataTable
    const gradeLevels = params.filters?.grade_level ?? [];
    for (const gl of gradeLevels) {
        searchParams.append("grade_level", gl);
    }
    const streamIds = params.filters?.stream_id ?? [];
    for (const sid of streamIds) {
        searchParams.append("stream_id", sid);
    }
    const academicYearIds = params.filters?.academic_year_id ?? [];
    for (const ayid of academicYearIds) {
        searchParams.append("academic_year_id", ayid);
    }

    if (params.academic_year_id) searchParams.set("academic_year_id", params.academic_year_id);
    if (params.academic_term_id) searchParams.set("academic_term_id", params.academic_term_id);
    if (params.grade_level) searchParams.set("grade_level", params.grade_level);
    if (params.stream_id) searchParams.set("stream_id", params.stream_id);
    if (params.search) searchParams.set("search", params.search);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ClassListResult>(`/api/v1/classes?${qs}`);
}
