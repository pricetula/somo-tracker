/**
 * TanStack Query hooks for the parents listing page.
 *
 * Uses the dedicated parents API module.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listParents,
    getParentDetail,
    updateParent,
    deleteParent,
    type ListParentsResponse,
    type Parent,
    type UpdateParentPayload,
} from "@/lib/api/parents";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const parentsKeys = {
    all: ["parents"] as const,
    list: (params?: Record<string, unknown>) => ["parents", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch parents with pagination and optional filters. */
export function useParents(
    opts: {
        page?: number;
        limit?: number;
        search?: string;
        includeInactive?: boolean;
        enabled?: boolean;
        /** Filter values keyed by FilterItem id, e.g. { education_level: ["Early_Years"] } */
        filters?: Record<string, string[]>;
    } = {}
) {
    const { page = 1, limit = 50, search, includeInactive = false, filters, enabled = true } = opts;

    return useQuery<ListParentsResponse>({
        queryKey: [...parentsKeys.list({ page, limit, search, includeInactive, filters })],
        queryFn: () =>
            listParents({
                page,
                limit,
                search,
                filters,
            }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single parent by ID. */
export function useParentDetail(userId: string | undefined) {
    return useQuery({
        queryKey: [...parentsKeys.all, "detail", userId],
        queryFn: () => getParentDetail(userId!),
        enabled: !!userId,
    });
}

/** Update a parent's profile with optimistic update. */
export function useUpdateParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, payload }: { userId: string; payload: UpdateParentPayload }) =>
            updateParent(userId, payload),
        onMutate: async ({ userId, payload }) => {
            await queryClient.cancelQueries({ queryKey: parentsKeys.all });
            const previousQueries = queryClient.getQueriesData<ListParentsResponse>({
                queryKey: parentsKeys.all,
            });

            queryClient.setQueriesData<ListParentsResponse>(
                { queryKey: parentsKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((p) => (p.id === userId ? { ...p, ...payload } : p)),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: (respData, err, val) => {
            if (respData && !err && val.userId) {
                queryClient.invalidateQueries({
                    queryKey: [...parentsKeys.all, "detail", val.userId],
                });
            }
        },
    });
}

/** Hard-delete a parent with optimistic removal. */
export function useDeleteParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => deleteParent(userId),
        onMutate: async (userId) => {
            await queryClient.cancelQueries({ queryKey: parentsKeys.all });
            const previousQueries = queryClient.getQueriesData<ListParentsResponse>({
                queryKey: parentsKeys.all,
            });

            queryClient.setQueriesData<ListParentsResponse>(
                { queryKey: parentsKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== userId),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _userId, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: parentsKeys.all });
        },
    });
}

/** Toggle parent active status with optimistic update (uses updateParent under the hood). */
export function useToggleParentActive() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, isActive }: { userId: string; isActive: boolean }) =>
            updateParent(userId, { is_active: isActive }),
        onMutate: async ({ userId, isActive }) => {
            await queryClient.cancelQueries({ queryKey: parentsKeys.all });
            const previousQueries = queryClient.getQueriesData<ListParentsResponse>({
                queryKey: parentsKeys.all,
            });

            queryClient.setQueriesData<ListParentsResponse>(
                { queryKey: parentsKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((p) =>
                            p.id === userId ? { ...p, is_active: isActive } : p
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: parentsKeys.all });
        },
    });
}
