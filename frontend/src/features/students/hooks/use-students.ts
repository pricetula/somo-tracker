/**
 * TanStack Query hook for listing students.
 *
 * Maps to GET /api/v1/students/list.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listStudents } from "../services/students-api";
import type { ListStudentsParams } from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const studentKeys = {
    all: ["students"] as const,
    list: (params: ListStudentsParams) => ["students", "list", params] as const,
};

// ─── Hook ─────────────────────────────────────────────────────────────────

/**
 * Fetch paginated student list.
 *
 * Supports optional search, class_id, and gender filters.
 */
export function useStudents(params: ListStudentsParams = {}, opts: { enabled?: boolean } = {}) {
    const { page = 1, limit = 50, search, class_id, gender } = params;
    const { enabled = true } = opts;

    return useQuery({
        queryKey: studentKeys.list({ page, limit, search, class_id, gender }),
        queryFn: () => listStudents({ page, limit, search, class_id, gender }),
        placeholderData: (prev) => prev,
        enabled,
    });
}
