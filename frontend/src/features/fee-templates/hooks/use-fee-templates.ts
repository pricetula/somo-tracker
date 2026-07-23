/**
 * TanStack Query hooks for fee templates management.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listFeeTemplates,
    createFeeTemplate,
    updateFeeTemplate,
    deleteFeeTemplate,
} from "@/lib/api/billing";
import { getErrorMessage } from "@/lib/errors";
import { STALE_TIMES } from "@/lib/query-config";
import type { CreateFeeTemplatePayload, UpdateFeeTemplatePayload } from "@/lib/api/billing";

// ─── Query keys ───────────────────────────────────────────────────────────

export const feeTemplateKeys = {
    all: ["fee-templates"] as const,
    lists: (filters?: { academic_term_id?: string; grade_level?: string }) =>
        [...feeTemplateKeys.all, "list", filters] as const,
};

// ─── Hooks — Queries ──────────────────────────────────────────────────────

export function useFeeTemplates(filters: { academic_term_id?: string; grade_level?: string } = {}) {
    return useQuery({
        queryKey: feeTemplateKeys.lists(filters),
        queryFn: () => listFeeTemplates(filters),
        staleTime: STALE_TIMES.REFERENCE_DATA,
        placeholderData: (prev) => prev,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

export function useCreateFeeTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateFeeTemplatePayload) => createFeeTemplate(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: feeTemplateKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: feeTemplateKeys.all,
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
            queryClient.invalidateQueries({ queryKey: feeTemplateKeys.all });
        },
    });
}

export function useUpdateFeeTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateFeeTemplatePayload }) =>
            updateFeeTemplate(id, payload),
        onMutate: async ({ id, payload }) => {
            await queryClient.cancelQueries({ queryKey: feeTemplateKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: feeTemplateKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: feeTemplateKeys.all },
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
            queryClient.invalidateQueries({ queryKey: feeTemplateKeys.all });
        },
    });
}

export function useDeleteFeeTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteFeeTemplate(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: feeTemplateKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: feeTemplateKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: feeTemplateKeys.all },
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
            queryClient.invalidateQueries({ queryKey: feeTemplateKeys.all });
        },
    });
}
