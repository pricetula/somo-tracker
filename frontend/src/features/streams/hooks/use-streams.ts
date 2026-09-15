/**
 * Streams feature — hooks for listing and creation.
 */

"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import {
    listStreams,
    createStreams,
    getStream,
    updateStream,
    deleteStreams,
} from "../services/api";
import type { Stream, CreateStreamsResponse } from "../types/stream";
import { useMeSession } from "@/features/auth/hooks/use-me-session";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const streamsKeys = {
    list: ["streams", "list"] as const,
    create: ["streams", "create"] as const,
    detail: (id: string) => ["streams", "detail", id] as const,
    multiDelete: ["streams", "multiDelete"] as const,
};

/**
 * Hook to fetch available streams for the active school.
 * Returns stream objects with id, name, color, etc.
 */
export function useStreams() {
    const { data: me } = useMeSession();

    return useQuery<Stream[], Error>({
        queryKey: streamsKeys.list,
        queryFn: async () => {
            // Ensure we have an active school
            if (!me?.active_school_id) {
                return [];
            }
            return listStreams();
        },
        enabled: !!me?.active_school_id, // Only run query if we have an active school
        retry: 0,
    });
}

/**
 * Hook to create new streams.
 */
export function useCreateStreams() {
    const { data: me } = useMeSession();

    return useMutation<CreateStreamsResponse, Error, string[]>({
        mutationKey: streamsKeys.create,
        mutationFn: async (names) => {
            // Only create if we have an active school
            if (!me?.active_school_id) {
                throw new Error("No active school");
            }
            return createStreams(names);
        },
        onSuccess: (data) => {
            toast.success(data.message ?? "Streams created successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useStream(id: string) {
    return useQuery<Stream, Error>({
        queryKey: streamsKeys.detail(id),
        queryFn: async () => {
            if (!id) throw new Error("Stream id required");
            return getStream(id);
        },
        enabled: !!id,
        retry: 0,
    });
}

export function useUpdateStream() {
    return useMutation<
        Stream,
        Error,
        { id: string; data: { name?: string; color?: string | null } }
    >({
        mutationKey: streamsKeys.create,
        mutationFn: async ({ id, data }) => updateStream(id, data),
        onSuccess: (_data) => {
            toast.success("Stream updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useDeleteStreams() {
    return useMutation<void, Error, string[]>({
        mutationKey: streamsKeys.multiDelete,
        mutationFn: async (ids) => deleteStreams(ids),
        onSuccess: () => {
            toast.success("Streams deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
