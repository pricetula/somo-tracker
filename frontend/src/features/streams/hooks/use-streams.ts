/**
 * useStreams — TanStack Query hooks for managing streams.
 *
 * Moved from settings-school to its own streams feature module so all
 * consumers import streams from one canonical place.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listStreams,
    createStream,
    updateStream,
    deleteStream,
    type StreamListResult,
} from "@/lib/api/streams";
import { getErrorMessage } from "@/lib/errors";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const streamKeys = {
    all: ["streams"] as const,
    list: () => [...streamKeys.all, "list"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch all streams for the active school. */
export function useStreamList() {
    return useQuery({
        queryKey: streamKeys.list(),
        queryFn: () => listStreams(),
        staleTime: STALE_TIMES.REFERENCE_DATA,
        placeholderData: (prev) => prev,
    });
}

/** Create a new stream with optimistic update. */
export function useCreateStream() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ name, color }: { name: string; color: string }) =>
            createStream({ name, color }),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: streamKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: streamKeys.all,
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
            void queryClient.invalidateQueries({ queryKey: streamKeys.all });
        },
    });
}

/** Update an existing stream's name and color with optimistic update. */
export function useUpdateStream() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, name, color }: { id: string; name: string; color: string }) =>
            updateStream(id, { name, color }),
        onMutate: async ({ id, name, color }) => {
            await queryClient.cancelQueries({ queryKey: streamKeys.all });
            const previousQueries = queryClient.getQueriesData<StreamListResult>({
                queryKey: streamKeys.all,
            });

            queryClient.setQueriesData<StreamListResult>({ queryKey: streamKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.map((s) => (s.id === id ? { ...s, name, color } : s)),
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
            void queryClient.invalidateQueries({ queryKey: streamKeys.all });
        },
    });
}

/** Delete a stream with optimistic removal. */
export function useDeleteStream() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteStream(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: streamKeys.all });
            const previousQueries = queryClient.getQueriesData<StreamListResult>({
                queryKey: streamKeys.all,
            });

            queryClient.setQueriesData<StreamListResult>({ queryKey: streamKeys.all }, (old) => {
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
            void queryClient.invalidateQueries({ queryKey: streamKeys.all });
        },
    });
}
