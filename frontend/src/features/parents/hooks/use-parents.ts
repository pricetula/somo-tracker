/**
 * TanStack Query hooks for the Parents feature.
 *
 * Covers parent CRUD, student linking/unlinking.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listParents,
    createParent,
    getParentDetail,
    getMyParentProfile,
    updateParent,
    deleteParent,
    linkStudent,
    unlinkStudent,
} from "@/lib/api/parents";
import { getErrorMessage } from "@/lib/errors";
import type {
    ListParentsResponse,
    ParentDetailResponse,
    CreateParentPayload,
    UpdateParentPayload,
    LinkStudentPayload,
    Parent,
} from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const parentKeys = {
    all: ["parents"] as const,
    list: (params?: Record<string, unknown>) => [...parentKeys.all, "list", params] as const,
    detail: (id: string) => [...parentKeys.all, "detail", id] as const,
};

// ─── Hooks: Parents List ─────────────────────────────────────────────────

/**
 * Fetch all parents as a Record<id, Parent> for O(1) lookups.
 *
 * Separate query key from the paginated useParents — fetches with a
 * generous limit. Best for lookup tables and cross-references.
 */
export function useParentMap() {
    return useQuery({
        queryKey: [...parentKeys.all, "map"] as const,
        queryFn: () => listParents({ limit: 500 }),
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
        select: (data: ListParentsResponse): Record<string, Parent> =>
            data?.items?.reduce?.(
                (acc, item) => {
                    acc[item.id] = item;
                    return acc;
                },
                {} as Record<string, Parent>
            ) ?? {},
    });
}

/** Fetch parents list, optionally filtered by search, student_id, or curriculum filters (education_level, grade_level), with pagination. */
export function useParents(
    params: {
        search?: string;
        student_id?: string;
        page?: number;
        limit?: number;
        filters?: Record<string, string[]>;
    } = {},
    opts: { enabled?: boolean } = {}
) {
    const { page = 1, limit = 50, search, student_id, filters } = params;
    const { enabled = true } = opts;

    return useQuery<ListParentsResponse>({
        queryKey: parentKeys.list({ page, limit, search, student_id, filters }),
        queryFn: () => listParents({ page, limit, search, student_id, filters }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single parent detail (with linked students). */
export function useParentDetail(id: string, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<ParentDetailResponse>({
        queryKey: parentKeys.detail(id),
        queryFn: () => getParentDetail(id),
        enabled: enabled && !!id,
    });
}

/** Fetch the authenticated parent's own profile with linked children. */
export function useMyParentProfile(opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<ParentDetailResponse>({
        queryKey: [...parentKeys.all, "me"] as const,
        queryFn: () => getMyParentProfile(),
        enabled,
    });
}

// ─── Mutations ────────────────────────────────────────────────────────────

/** Create a parent with optimistic update. */
export function useCreateParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateParentPayload) => createParent(data),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: parentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: parentKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _data, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
        },
    });
}

/** Update a parent (phone_number, is_active) with optimistic update. */
export function useUpdateParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateParentPayload }) =>
            updateParent(id, data),
        onMutate: async ({ id, data }) => {
            await queryClient.cancelQueries({ queryKey: parentKeys.all });
            const previousQueries = queryClient.getQueriesData<ListParentsResponse>({
                queryKey: parentKeys.all,
            });

            queryClient.setQueriesData<ListParentsResponse>({ queryKey: parentKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.map((p) => (p.id === id ? { ...p, ...data } : p)),
                };
            });

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
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
        },
    });
}

/** Delete a parent with optimistic removal. */
export function useDeleteParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteParent(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: parentKeys.all });
            const previousQueries = queryClient.getQueriesData<ListParentsResponse>({
                queryKey: parentKeys.all,
            });

            queryClient.setQueriesData<ListParentsResponse>({ queryKey: parentKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.filter((item) => item.id !== id),
                };
            });

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
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
        },
    });
}

/** Link a student to a parent with optimistic update. */
export function useLinkStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ parentId, data }: { parentId: string; data: LinkStudentPayload }) =>
            linkStudent(parentId, data),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: parentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: parentKeys.all,
            });
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
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
        },
    });
}

/** Unlink a student from a parent with optimistic update. */
export function useUnlinkStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ parentId, studentId }: { parentId: string; studentId: string }) =>
            unlinkStudent(parentId, studentId),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: parentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: parentKeys.all,
            });
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
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
        },
    });
}
