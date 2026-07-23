/**
 * TanStack Query hooks for import jobs management.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listJobs,
    getActiveImportJob,
    getImportJob,
    getImportFailures,
    cancelImportJob,
} from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";

// ─── Query keys ───────────────────────────────────────────────────────────

export const importJobKeys = {
    all: ["import-jobs"] as const,
    lists: () => [...importJobKeys.all, "list"] as const,
    active: () => [...importJobKeys.all, "active"] as const,
    detail: (id: string) => [...importJobKeys.all, "detail", id] as const,
    failures: (id: string) => [...importJobKeys.all, "failures", id] as const,
};

// ─── Hooks — Queries ──────────────────────────────────────────────────────

/** Fetch paginated import jobs for the active school. */
export function useImportJobs(params: { page?: number; limit?: number } = {}) {
    return useQuery({
        queryKey: [...importJobKeys.lists(), params],
        queryFn: () => listJobs(params),
        staleTime: 30 * 1000,
        placeholderData: (prev) => prev,
    });
}

/** Fetch the currently active import job (if any). */
export function useActiveImportJob() {
    return useQuery({
        queryKey: importJobKeys.active(),
        queryFn: () => getActiveImportJob(),
        staleTime: 15 * 1000,
        refetchInterval: (query) => {
            const data = query.state.data;
            if (!data?.active || !data.job) return false;
            // Poll while job is in a non-terminal state
            const terminalStatuses = ["completed", "completed_with_errors", "failed", "cancelled"];
            if (terminalStatuses.includes(data.job.status)) return false;
            return 5_000; // poll every 5s while active
        },
    });
}

/** Fetch a single import job by ID. */
export function useImportJobDetail(jobId: string) {
    return useQuery({
        queryKey: importJobKeys.detail(jobId),
        queryFn: () => getImportJob(jobId),
        staleTime: 15 * 1000,
        refetchInterval: (query) => {
            const job = query.state.data;
            if (!job) return false;
            const terminalStatuses = ["completed", "completed_with_errors", "failed", "cancelled"];
            if (terminalStatuses.includes(job.status)) return false;
            return 5_000;
        },
    });
}

/** Fetch failures for a completed import job. */
export function useImportJobFailures(
    jobId: string,
    params: { page?: number; limit?: number } = {}
) {
    return useQuery({
        queryKey: [...importJobKeys.failures(jobId), params],
        queryFn: () => getImportFailures(jobId, params),
        staleTime: 60 * 1000,
        enabled: !!jobId,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

/** Cancel a running import job with optimistic update. */
export function useCancelImportJob() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (jobId: string) => cancelImportJob(jobId),
        onMutate: async (jobId) => {
            await queryClient.cancelQueries({ queryKey: importJobKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: importJobKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string; status?: string }[] }>(
                { queryKey: importJobKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((job) =>
                            job.id === jobId ? { ...job, status: "cancelled" } : job
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _jobId, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: importJobKeys.all });
        },
    });
}
