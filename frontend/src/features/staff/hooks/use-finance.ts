/**
 * TanStack Query hooks for the finance staff listing page.
 *
 * Uses its own query key and API module — does not re-use the generic
 * members hooks.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listFinanceStaff, toggleFinanceActive, type ListMembersResponse } from "@/lib/api/finance";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const financeKeys = {
    all: ["finance"] as const,
    list: (params?: Record<string, unknown>) => ["finance", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch finance staff (FINANCE role). */
export function useFinanceStaff(
    opts: {
        page?: number;
        limit?: number;
        search?: string;
        includeInactive?: boolean;
        enabled?: boolean;
    } = {}
) {
    const { page = 1, limit = 50, search, includeInactive = false, enabled = true } = opts;

    return useQuery<ListMembersResponse>({
        queryKey: [...financeKeys.list({ page, limit, search, includeInactive })],
        queryFn: () =>
            listFinanceStaff({ page, per_page: limit, search, include_inactive: includeInactive }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Toggle finance staff active status with optimistic update. */
export function useToggleFinanceActive() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, isActive }: { userId: string; isActive: boolean }) =>
            toggleFinanceActive(userId, isActive),
        onMutate: async ({ userId, isActive }) => {
            await queryClient.cancelQueries({ queryKey: financeKeys.all });
            const previousQueries = queryClient.getQueriesData<ListMembersResponse>({
                queryKey: financeKeys.all,
            });

            queryClient.setQueriesData<ListMembersResponse>(
                { queryKey: financeKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        members: old.members.map((m) =>
                            m.id === userId ? { ...m, is_active: isActive } : m
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
            queryClient.invalidateQueries({ queryKey: financeKeys.all });
        },
    });
}
