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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: feeTemplateKeys.all });
            toast.success("Fee template created");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useUpdateFeeTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateFeeTemplatePayload }) =>
            updateFeeTemplate(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: feeTemplateKeys.all });
            toast.success("Fee template updated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useDeleteFeeTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteFeeTemplate(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: feeTemplateKeys.all });
            toast.success("Fee template deleted");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}
