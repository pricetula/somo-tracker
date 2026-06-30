/**
 * TanStack Query hooks for the nurses listing page.
 *
 * Uses its own query key and API module — does not re-use the generic
 * members hooks.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listNurses, toggleNurseActive, type ListMembersResponse } from "@/lib/api/nurses";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const nursesKeys = {
    all: ["nurses"] as const,
    list: (params?: Record<string, unknown>) => ["nurses", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch nurses (NURSE role). */
export function useNurses(
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
        queryKey: [...nursesKeys.list({ page, limit, search, includeInactive })],
        queryFn: () =>
            listNurses({ page, per_page: limit, search, include_inactive: includeInactive }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Toggle nurse active status with optimistic update. */
export function useToggleNurseActive() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, isActive }: { userId: string; isActive: boolean }) =>
            toggleNurseActive(userId, isActive),
        onMutate: async ({ userId, isActive }) => {
            await queryClient.cancelQueries({ queryKey: nursesKeys.all });
            const previousQueries = queryClient.getQueriesData<ListMembersResponse>({
                queryKey: nursesKeys.all,
            });

            queryClient.setQueriesData<ListMembersResponse>({ queryKey: nursesKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    members: old.members.map((m) =>
                        m.id === userId ? { ...m, is_active: isActive } : m
                    ),
                };
            });

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
            queryClient.invalidateQueries({ queryKey: nursesKeys.all });
        },
    });
}
