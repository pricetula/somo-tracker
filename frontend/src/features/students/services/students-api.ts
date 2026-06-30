/**
 * Students API functions.
 *
 * Endpoints:
 *   GET /api/v1/students/list — paginated student listing
 */

import { api } from "@/lib/api/client";
import type { ListStudentsResponse, ListStudentsParams } from "../types";

/**
 * List students with pagination and optional filters.
 *
 * Maps to GET /api/v1/students/list.
 */
export async function listStudents(params: ListStudentsParams = {}): Promise<ListStudentsResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    if (params.class_id) searchParams.set("class_id", params.class_id);
    if (params.gender) searchParams.set("gender", params.gender);

    const qs = searchParams.toString();
    return api.get<ListStudentsResponse>(`/api/v1/students/list?${qs}`);
}
