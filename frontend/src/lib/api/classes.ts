/**
 * Classes API client.
 *
 * Endpoints:
 *   GET  /api/v1/schools/classes            → Class[] (with filter query params)
 *   GET  /api/v1/schools/classes/grades     → Grade[]
 *   POST /api/v1/schools/classes/generate    → GenerateResult
 */

import { api } from "@/lib/api/client";
import type {
    ClassItem,
    ClassListParams,
    Grade,
    GeneratePayload,
    GenerateResult,
} from "@/features/classes/types";

/**
 * Fetch classes with optional filters.
 * Supports grade_ids (comma-separated), search (ILIKE on name), and is_active.
 */
export async function fetchClasses(params?: ClassListParams): Promise<ClassItem[]> {
    const searchParams = new URLSearchParams();
    if (params?.grade_ids && params.grade_ids.length > 0) {
        searchParams.set("grade_ids", params.grade_ids.join(","));
    }
    if (params?.search) {
        searchParams.set("search", params.search);
    }
    if (params?.is_active !== undefined) {
        searchParams.set("is_active", String(params.is_active));
    }

    const qs = searchParams.toString();

    try {
        return await api.get<ClassItem[]>(`/api/v1/schools/classes${qs ? `?${qs}` : ""}`);
    } catch (err) {
        // On 404, return empty list (no classes configured)
        if (
            err &&
            typeof err === "object" &&
            "status" in err &&
            (err as { status: number }).status === 404
        ) {
            return [];
        }
        throw err;
    }
}

/** Fetch all grades for the school's education system. */
export async function fetchGrades(): Promise<Grade[]> {
    return await api.get<Grade[]>("/api/v1/schools/classes/grades");
}

/** Generate (bulk-create) classrooms from stream names × grade levels. */
export async function generateClasses(payload: GeneratePayload): Promise<GenerateResult> {
    return await api.post<GenerateResult>("/api/v1/schools/classes/generate", payload);
}
