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
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

export function useCreateFeeCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateFeeCategoryPayload) => createFeeCategory(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: feeCategoryKeys.all });
            toast.success("Fee category created");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useUpdateFeeCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateFeeCategoryPayload }) =>
            updateFeeCategory(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: feeCategoryKeys.all });
            toast.success("Fee category updated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useDeleteFeeCategory() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteFeeCategory(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: feeCategoryKeys.all });
            toast.success("Fee category deleted");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}
