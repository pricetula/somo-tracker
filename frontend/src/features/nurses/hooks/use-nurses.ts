/**
 * TanStack Query hooks for the nurses listing page.
 *
 * Uses the generic members API for detail/update since nurses
 * are managed through the shared /api/v1/members endpoints.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listNurses,
    toggleNurseActive,
    deleteNurse,
    type ListMembersResponse,
} from "@/lib/api/nurses";
import type { Member } from "@/lib/api/generated";
import { getMember, updateMember } from "@/lib/api/members";

import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const nursesKeys = {
    all: ["nurses"] as const,
    list: (params?: Record<string, unknown>) => ["nurses", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/**
 * Fetch all nurses as a Record<id, Member> for O(1) lookups.
 *
 * Separate query key from the paginated useNurses — fetches with a
 * generous limit. Best for lookup tables and cross-references.
 */
export function useNurseMap() {
    return useQuery({
        queryKey: [...nursesKeys.all, "map"] as const,
        queryFn: () => listNurses({ limit: 500 }),
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
        select: (data: ListMembersResponse): Record<string, Member> =>
            data?.items?.reduce?.(
                (acc, item) => {
                    acc[item.id] = item;
                    return acc;
                },
                {} as Record<string, Member>
            ) ?? {},
    });
}

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
        queryFn: () => listNurses({ page, limit, search, include_inactive: includeInactive }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single nurse by ID (uses generic members API). */
export function useNurseDetail(userId: string | undefined) {
    return useQuery({
        queryKey: [...nursesKeys.all, "detail", userId],
        queryFn: () => getMember(userId!),
        enabled: !!userId,
    });
}

/** Update a nurse's profile (uses generic members API). */
export function useUpdateNurse() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, payload }: { userId: string; payload: { full_name?: string } }) =>
            updateMember(userId, payload as { full_name: string }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: nursesKeys.all });
            toast.success("Nurse updated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Hard-delete a nurse member. */
export function useDeleteNurse() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => deleteNurse(userId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: nursesKeys.all });
            toast.success("Nurse deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
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
                    items: old.items.map((m) =>
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
