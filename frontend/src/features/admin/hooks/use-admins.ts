/**
 * TanStack Query hooks for the admins listing page.
 *
 * Uses its own query key and API module.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listAdmins,
    toggleAdminActive,
    deleteAdmin,
    type ListMembersResponse,
    type Member,
} from "@/lib/api/admins";
import { getMember, updateMember } from "@/lib/api/members";

import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const adminsKeys = {
    all: ["admins"] as const,
    list: (params?: Record<string, unknown>) => ["admins", "list", params] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch admins (SCHOOL_ADMIN role). */
export function useAdmins(
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
        queryKey: [...adminsKeys.list({ page, limit, search, includeInactive })],
        queryFn: () => listAdmins({ page, limit, search, include_inactive: includeInactive }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single admin by ID. */
export function useAdminDetail(userId: string | undefined) {
    return useQuery({
        queryKey: [...adminsKeys.all, "detail", userId],
        queryFn: () => getMember(userId!),
        enabled: !!userId,
    });
}

/** Update an admin's profile with optimistic update. */
export function useUpdateAdmin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, payload }: { userId: string; payload: { full_name?: string } }) =>
            updateMember(userId, payload as { full_name: string }),
        onMutate: async ({ userId, payload }) => {
            await queryClient.cancelQueries({ queryKey: adminsKeys.all });
            const previousQueries = queryClient.getQueriesData<ListMembersResponse>({
                queryKey: adminsKeys.all,
            });

            queryClient.setQueriesData<{ pages: Array<{ items: Array<Member> }> }>(
                { queryKey: adminsKeys.all },
                (old) => {
                    if (!old?.pages) return old;
                    // Build index once: O(N)
                    const userIndex = new Map();

                    old.pages.forEach((page) => {
                        page.items.forEach((item) => userIndex.set(item.id, item));
                    });

                    // Direct O(1) lookup & update
                    const target = userIndex.get(userId);
                    if (target) {
                        target.full_name = payload.full_name;
                    }

                    return {
                        ...old,
                        pages: old?.pages,
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
    });
}

/** Hard-delete an admin member with optimistic removal. */
export function useDeleteAdmin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => deleteAdmin(userId),
        onMutate: async (userId) => {
            await queryClient.cancelQueries({ queryKey: adminsKeys.all });
            const previousQueries = queryClient.getQueriesData<ListMembersResponse>({
                queryKey: adminsKeys.all,
            });

            queryClient.setQueriesData<ListMembersResponse>({ queryKey: adminsKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.filter((item) => item.id !== userId),
                };
            });

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
            queryClient.invalidateQueries({ queryKey: adminsKeys.all });
        },
    });
}

/** Toggle admin active status with optimistic update. */
export function useToggleAdminActive() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, isActive }: { userId: string; isActive: boolean }) =>
            toggleAdminActive(userId, isActive),
        onMutate: async ({ userId, isActive }) => {
            await queryClient.cancelQueries({ queryKey: adminsKeys.all });
            const previousQueries = queryClient.getQueriesData<ListMembersResponse>({
                queryKey: adminsKeys.all,
            });

            queryClient.setQueriesData<ListMembersResponse>({ queryKey: adminsKeys.all }, (old) => {
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
            queryClient.invalidateQueries({ queryKey: adminsKeys.all });
        },
    });
}
