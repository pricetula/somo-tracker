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
} from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const parentKeys = {
    all: ["parents"] as const,
    list: (params?: Record<string, unknown>) => [...parentKeys.all, "list", params] as const,
    detail: (id: string) => [...parentKeys.all, "detail", id] as const,
};

// ─── Hooks: Parents List ─────────────────────────────────────────────────

/** Fetch parents list, optionally filtered by search or student_id. */
export function useParents(
    params: { search?: string; student_id?: string } = {},
    opts: { enabled?: boolean } = {}
) {
    const { enabled = true } = opts;

    return useQuery<ListParentsResponse>({
        queryKey: parentKeys.list(params),
        queryFn: () => listParents(params),
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
