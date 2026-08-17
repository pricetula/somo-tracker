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
import { STALE_TIMES } from "@/lib/query-config";
import type { ListStudentsParams, ListStudentsResponse, Student } from "@/lib/api/students";

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

/**
 * Fetch all students as a Record<id, Student> for O(1) lookups.
 *
 * Separate query key from the paginated useStudents — fetches with a
 * generous limit. Best for lookup tables and cross-references.
 */
export function useStudentMap() {
    return useQuery({
        queryKey: [...studentKeys.all, "map"] as const,
        queryFn: () => listStudents({ limit: 500 }),
        staleTime: STALE_TIMES.REFERENCE_DATA,
        placeholderData: (prev) => prev,
        select: (data: ListStudentsResponse): Record<string, Student> =>
            data?.items?.reduce?.(
                (acc, item) => {
                    acc[item.id] = item;
                    return acc;
                },
                {} as Record<string, Student>
            ) ?? {},
    });
}

/** Hard-delete a student with optimistic removal. */
export function useDeleteStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteStudent(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: studentKeys.all });
            const previousQueries = queryClient.getQueriesData<ListStudentsResponse>({
                queryKey: studentKeys.all,
            });

            queryClient.setQueriesData<ListStudentsResponse>(
                { queryKey: studentKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
        },
    });
}
