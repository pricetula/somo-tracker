/**
 * TanStack Query hooks for the finance staff listing page.
 *
 * Uses the generic members API for detail/update since finance staff
 * are managed through the shared /api/v1/members endpoints.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listFinanceStaff,
    toggleFinanceActive,
    deleteFinanceStaff,
    type ListMembersResponse,
} from "@/lib/api/finance";
import { getMember, updateMember } from "@/lib/api/members";

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
        queryFn: () => listFinanceStaff({ page, limit, search, include_inactive: includeInactive }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single finance staff member by ID (uses generic members API). */
export function useFinanceDetail(userId: string | undefined) {
    return useQuery({
        queryKey: [...financeKeys.all, "detail", userId],
        queryFn: () => getMember(userId!),
        enabled: !!userId,
    });
}

/** Update a finance staff member's profile (uses generic members API). */
export function useUpdateFinance() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, payload }: { userId: string; payload: { full_name?: string } }) =>
            updateMember(userId, payload as { full_name: string }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: financeKeys.all });
            toast.success("Finance staff updated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Hard-delete a finance staff member. */
export function useDeleteFinanceStaff() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => deleteFinanceStaff(userId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: financeKeys.all });
            toast.success("Finance staff deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
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
                        items: old.items.map((m) =>
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
