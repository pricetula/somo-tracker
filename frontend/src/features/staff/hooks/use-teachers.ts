/**
 * TanStack Query hooks for the teachers listing page.
 *
 * Uses its own query key and API module — does not re-use the generic
 * members hooks. Maps to the dedicated teachers API.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listTeachers, toggleTeacherActive, type ListTeachersResponse } from "@/lib/api/teachers";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const teachersKeys = {
    all: ["teachers"] as const,
    list: (params?: Record<string, unknown>) => ["teachers", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch teachers with extended fields (TSC, KNEC, teacher_role). */
export function useTeachers(
    opts: {
        page?: number;
        limit?: number;
        search?: string;
        includeInactive?: boolean;
        enabled?: boolean;
    } = {}
) {
    const { page = 1, limit = 50, search, includeInactive = false, enabled = true } = opts;

    return useQuery<ListTeachersResponse>({
        queryKey: [...teachersKeys.list({ page, limit, search, includeInactive })],
        queryFn: () =>
            listTeachers({ page, per_page: limit, search, include_inactive: includeInactive }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Toggle teacher active status with optimistic update. */
export function useToggleTeacherActive() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, isActive }: { userId: string; isActive: boolean }) =>
            toggleTeacherActive(userId, isActive),
        onMutate: async ({ userId, isActive }) => {
            await queryClient.cancelQueries({ queryKey: teachersKeys.all });
            const previousQueries = queryClient.getQueriesData<ListTeachersResponse>({
                queryKey: teachersKeys.all,
            });

            queryClient.setQueriesData<ListTeachersResponse>(
                { queryKey: teachersKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        teachers: old.teachers.map((t) =>
                            t.id === userId ? { ...t, is_active: isActive } : t
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            // Rollback optimistic update
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: teachersKeys.all });
        },
    });
}
