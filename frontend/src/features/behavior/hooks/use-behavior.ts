/**
 * useBehavior — TanStack Query hooks for behavior operations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listBehaviorCategories,
    createBehaviorCategory,
    updateBehaviorCategory,
    createBehaviorNote,
    getBehaviorPendingQueue,
    reviewBehaviorNote,
    listTeacherNotes,
    type CreateCategoryPayload,
    type UpdateCategoryPayload,
    type CreateNotePayload,
    type ReviewDecisionPayload,
    type TeacherNotesResponse,
} from "@/lib/api/behavior";
import { deleteBehaviorNote } from "@/lib/api/behavior";
import { getErrorMessage } from "@/lib/errors";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const behaviorKeys = {
    all: ["behavior"] as const,
    categories: (activeOnly?: boolean) => [...behaviorKeys.all, "categories", activeOnly] as const,
    queue: () => [...behaviorKeys.all, "queue"] as const,
    note: (id: string) => [...behaviorKeys.all, "notes", id] as const,
};

// ─── Categories Hooks ─────────────────────────────────────────────────────

/** Fetch all behavior categories. */
export function useBehaviorCategories(activeOnly?: boolean) {
    return useQuery({
        queryKey: behaviorKeys.categories(activeOnly),
        queryFn: () => listBehaviorCategories(activeOnly),
        staleTime: STALE_TIMES.REFERENCE_DATA,
    });
}

/** Create a new behavior category. */
export function useCreateBehaviorCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateCategoryPayload) => createBehaviorCategory(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.categories() });
            toast.success("Category created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a behavior category. */
export function useUpdateBehaviorCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateCategoryPayload }) =>
            updateBehaviorCategory(id, payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.categories() });
            toast.success("Category updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Notes Hooks ──────────────────────────────────────────────────────────

/** Create a new behavior note (teacher action). */
export function useCreateBehaviorNote() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateNotePayload) => createBehaviorNote(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.queue() });
            toast.success("Note submitted for review");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Get the pending review queue (admin view). */
export function useBehaviorPendingQueue() {
    return useQuery({
        queryKey: behaviorKeys.queue(),
        queryFn: () => getBehaviorPendingQueue(),
        refetchInterval: 30_000,
        staleTime: STALE_TIMES.LIVE,
    });
}

/** Fetch the current user's (teacher's) own behavior notes. */
export function useTeacherNotes() {
    return useQuery<TeacherNotesResponse>({
        queryKey: [...behaviorKeys.all, "my-notes"],
        queryFn: () => listTeacherNotes(),
        staleTime: STALE_TIMES.FREQUENT,
    });
}

/** Delete a behavior note permanently. */
export function useDeleteBehaviorNote() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteBehaviorNote(id),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.all });
            toast.success("Behavior note deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Approve or reject a behavior note. */
export function useReviewBehaviorNote() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ noteId, payload }: { noteId: string; payload: ReviewDecisionPayload }) =>
            reviewBehaviorNote(noteId, payload),
        onSuccess: (data) => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.queue() });
            if (data.decision === "APPROVED") {
                toast.success("Note approved");
            } else {
                toast.success("Note rejected");
            }
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
