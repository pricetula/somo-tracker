/**
 * useStreams — TanStack Query hooks for managing streams.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { listStreams, createStream, updateStream, deleteStream } from "@/lib/api/streams";
import { getErrorMessage } from "@/lib/errors";
import type { Stream } from "../types";

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
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
        select: (data): { items: Stream[]; total: number } => ({
            items: data.items,
            total: data.total,
        }),
    });
}

/** Create a new stream. */
export function useCreateStream() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ name, color }: { name: string; color: string }) =>
            createStream({ name, color }),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: streamKeys.list() });
            toast.success("Stream created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update an existing stream's name and color. */
export function useUpdateStream() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, name, color }: { id: string; name: string; color: string }) =>
            updateStream(id, { name, color }),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: streamKeys.list() });
            toast.success("Stream updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a stream. */
export function useDeleteStream() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteStream(id),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: streamKeys.list() });
            toast.success("Stream deleted successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
