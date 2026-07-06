/**
 * TanStack Query hooks for the Students feature.
 *
 * Maps to GET /api/v1/students/list, DELETE /api/v1/students/:id.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { listStudents, deleteStudent } from "@/lib/api/students";
import { getErrorMessage } from "@/lib/errors";
import type { ListStudentsParams, ListStudentsResponse } from "@/lib/api/students";

// ─── Query keys ───────────────────────────────────────────────────────────

export const studentKeys = {
    all: ["students"] as const,
    list: (params: ListStudentsParams) => ["students", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/**
 * Fetch paginated student list.
 *
 * Supports optional search, class_id, gender, enrollment_status, and
 * curriculum filters (education_level, grade_level) via the `filters` object.
 */
export function useStudents(params: ListStudentsParams = {}, opts: { enabled?: boolean } = {}) {
    const { page = 1, limit = 50, search, class_id, gender, enrollment_status, filters } = params;
    const { enabled = true } = opts;

    return useQuery<ListStudentsResponse>({
        queryKey: studentKeys.list({
            page,
            limit,
            search,
            class_id,
            gender,
            enrollment_status,
            filters,
        }),
        queryFn: () =>
            listStudents({ page, limit, search, class_id, gender, enrollment_status, filters }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Hard-delete a student. */
export function useDeleteStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteStudent(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
            toast.success("Student deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
