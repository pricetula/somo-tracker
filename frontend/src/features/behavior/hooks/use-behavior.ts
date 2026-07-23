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

/** Create a new behavior category with optimistic update. */
export function useCreateBehaviorCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateCategoryPayload) => createBehaviorCategory(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: behaviorKeys.categories() });
            const previousQueries = queryClient.getQueriesData({
                queryKey: behaviorKeys.categories(),
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.categories() });
        },
    });
}

/** Update a behavior category with optimistic update. */
export function useUpdateBehaviorCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateCategoryPayload }) =>
            updateBehaviorCategory(id, payload),
        onMutate: async ({ id, payload }) => {
            await queryClient.cancelQueries({ queryKey: behaviorKeys.categories() });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: behaviorKeys.categories(),
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: behaviorKeys.categories() },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === id ? { ...item, ...payload } : item
                        ),
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
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.categories() });
        },
    });
}

// ─── Notes Hooks ──────────────────────────────────────────────────────────

/** Create a new behavior note (teacher action) with optimistic update. */
export function useCreateBehaviorNote() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateNotePayload) => createBehaviorNote(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: behaviorKeys.queue() });
            const previousQueries = queryClient.getQueriesData({
                queryKey: behaviorKeys.queue(),
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.queue() });
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

/** Delete a behavior note permanently with optimistic removal. */
export function useDeleteBehaviorNote() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteBehaviorNote(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: behaviorKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: behaviorKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: behaviorKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

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
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.all });
        },
    });
}

/** Approve or reject a behavior note with optimistic update. */
export function useReviewBehaviorNote() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ noteId, payload }: { noteId: string; payload: ReviewDecisionPayload }) =>
            reviewBehaviorNote(noteId, payload),
        onMutate: async ({ noteId, payload }) => {
            await queryClient.cancelQueries({ queryKey: behaviorKeys.queue() });
            const previousQueries = queryClient.getQueriesData<{
                items: { id: string; status?: string }[];
            }>({
                queryKey: behaviorKeys.queue(),
            });

            queryClient.setQueriesData<{ items: { id: string; status?: string }[] }>(
                { queryKey: behaviorKeys.queue() },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === noteId
                                ? {
                                      ...item,
                                      status:
                                          payload.decision === "APPROVED" ? "APPROVED" : "REJECTED",
                                      ...payload,
                                  }
                                : item
                        ),
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
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: behaviorKeys.queue() });
        },
    });
}
