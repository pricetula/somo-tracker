/**
 * TanStack Query hooks for fee categories management.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listFeeCategories,
    createFeeCategory,
    updateFeeCategory,
    deleteFeeCategory,
} from "@/lib/api/billing";
import { getErrorMessage } from "@/lib/errors";
import { STALE_TIMES } from "@/lib/query-config";
import type { CreateFeeCategoryPayload, UpdateFeeCategoryPayload } from "@/lib/api/billing";

// ─── Query keys ───────────────────────────────────────────────────────────

export const feeCategoryKeys = {
    all: ["fee-categories"] as const,
    lists: () => [...feeCategoryKeys.all, "list"] as const,
};

// ─── Hooks — Queries ──────────────────────────────────────────────────────

export function useFeeCategories() {
    return useQuery({
        queryKey: feeCategoryKeys.lists(),
        queryFn: () => listFeeCategories(),
        staleTime: STALE_TIMES.REFERENCE_DATA,
        placeholderData: (prev) => prev,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

export function useCreateFeeCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateFeeCategoryPayload) => createFeeCategory(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: feeCategoryKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: feeCategoryKeys.all,
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
            queryClient.invalidateQueries({ queryKey: feeCategoryKeys.all });
        },
    });
}

export function useUpdateFeeCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateFeeCategoryPayload }) =>
            updateFeeCategory(id, payload),
        onMutate: async ({ id, payload }) => {
            await queryClient.cancelQueries({ queryKey: feeCategoryKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: feeCategoryKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: feeCategoryKeys.all },
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
            queryClient.invalidateQueries({ queryKey: feeCategoryKeys.all });
        },
    });
}

export function useDeleteFeeCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteFeeCategory(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: feeCategoryKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: feeCategoryKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: feeCategoryKeys.all },
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
            queryClient.invalidateQueries({ queryKey: feeCategoryKeys.all });
        },
    });
}
