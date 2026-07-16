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

/** Update an admin's profile. */
export function useUpdateAdmin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, payload }: { userId: string; payload: { full_name?: string } }) =>
            updateMember(userId, payload as { full_name: string }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: adminsKeys.all });
            toast.success("Admin updated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Hard-delete an admin member. */
export function useDeleteAdmin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => deleteAdmin(userId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: adminsKeys.all });
            toast.success("Admin deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
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
