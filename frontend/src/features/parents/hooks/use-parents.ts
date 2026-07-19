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

/** Create a parent. */
export function useCreateParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateParentPayload) => createParent(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
            toast.success("Parent created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a parent (phone_number, is_active). */
export function useUpdateParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateParentPayload }) =>
            updateParent(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
            toast.success("Parent updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a parent. */
export function useDeleteParent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteParent(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
            toast.success("Parent deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Link a student to a parent. */
export function useLinkStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ parentId, data }: { parentId: string; data: LinkStudentPayload }) =>
            linkStudent(parentId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
            toast.success("Student linked");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Unlink a student from a parent. */
export function useUnlinkStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ parentId, studentId }: { parentId: string; studentId: string }) =>
            unlinkStudent(parentId, studentId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: parentKeys.all });
            toast.success("Student unlinked");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
