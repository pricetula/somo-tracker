/**
 * TanStack Query hooks for the teachers listing page.
 *
 * Uses its own query key and API module — does not re-use the generic
 * members hooks. Maps to the dedicated teachers API.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listTeachers,
    toggleTeacherActive,
    deleteTeacher,
    type ListTeachersResponse,
} from "@/lib/api/teachers";
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
        /** Filter values keyed by FilterItem id, e.g. { education_level: ["Early_Years"] } */
        filters?: Record<string, string[]>;
        enabled?: boolean;
    } = {}
) {
    const { page = 1, limit = 50, search, includeInactive = false, filters, enabled = true } = opts;

    return useQuery<ListTeachersResponse>({
        queryKey: [...teachersKeys.list({ page, limit, search, includeInactive, filters })],
        queryFn: () =>
            listTeachers({
                page,
                limit,
                search,
                include_inactive: includeInactive,
                filters,
            }),
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
                        items: old.items.map((t) =>
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

/** Hard-delete a teacher. */
export function useDeleteTeacher() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => deleteTeacher(userId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: teachersKeys.all });
            toast.success("Teacher deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
