/**
 * Classes API functions.
 *
 * Endpoints:
 *   GET  /api/v1/classes — list classes (requires academic_year_id and academic_term_id)
 *   POST /api/v1/classes — create class
 *   PUT  /api/v1/classes/:id — update class
 *   DELETE /api/v1/classes — bulk delete classes
 */

import { api } from "./client";
import type { Class, ClassListResult } from "./generated";

export type { Class, ClassListResult };

/** List classes for the active school. */
export async function listClasses(params: {
    academic_year_id: string;
    academic_term_id: string;
    grade_level?: string;
    stream_id?: string;
    page?: number;
    limit?: number;
}): Promise<ClassListResult> {
    const searchParams = new URLSearchParams();
    searchParams.set("academic_year_id", params.academic_year_id);
    searchParams.set("academic_term_id", params.academic_term_id);
    if (params.grade_level) searchParams.set("grade_level", params.grade_level);
    if (params.stream_id) searchParams.set("stream_id", params.stream_id);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ClassListResult>(`/api/v1/classes?${qs}`);
}
