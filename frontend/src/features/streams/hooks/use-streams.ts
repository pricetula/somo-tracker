/**
 * Streams feature — hooks for listing and creation.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
    const queryClient = useQueryClient();

    return useMutation<CreateStreamsResponse, Error, { name: string; color?: string }[]>({
        mutationKey: streamsKeys.create,
        mutationFn: async (items) => {
            if (!me?.active_school_id) {
                throw new Error("No active school");
            }
            return createStreams(items);
        },
        async onMutate(items) {
            await queryClient.cancelQueries({ queryKey: streamsKeys.list });
            const previousList = queryClient.getQueryData<Stream[]>(streamsKeys.list);
            const optimistic = items.map(
                (item, idx) =>
                    ({
                        id: `optimistic-${idx}`,
                        school_id: me?.active_school_id ?? "",
                        name: item.name,
                        color: item.color ?? "#0050d1",
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                    }) as Stream
            );
            if (previousList) {
                queryClient.setQueryData<Stream[]>(streamsKeys.list, [
                    ...previousList,
                    ...optimistic,
                ]);
            }
            return { previousList };
        },
        onError(err, _variables, context) {
            if (context?.previousList) {
                queryClient.setQueryData<Stream[]>(streamsKeys.list, context.previousList);
            }
            toast.error(getErrorMessage(err));
        },
        onSettled(_data, _err) {
            queryClient.invalidateQueries({ queryKey: streamsKeys.list });
        },
        onSuccess: (data) => {
            toast.success(data.message ?? "Streams created successfully");
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
    const queryClient = useQueryClient();
    return useMutation<
        Stream,
        Error,
        { id: string; data: { name?: string; color?: string | null } }
    >({
        mutationKey: streamsKeys.create,
        mutationFn: async ({ id, data }) => updateStream(id, data),
        async onMutate(variables) {
            await queryClient.cancelQueries({ queryKey: streamsKeys.detail(variables.id) });
            await queryClient.cancelQueries({ queryKey: streamsKeys.list });
            const previousStream = queryClient.getQueryData<Stream>(
                streamsKeys.detail(variables.id)
            );
            const previousList = queryClient.getQueryData<Stream[]>(streamsKeys.list);
            if (previousStream) {
                queryClient.setQueryData<Stream>(streamsKeys.detail(variables.id), {
                    ...previousStream,
                    ...variables.data,
                });
            }
            if (previousList) {
                queryClient.setQueryData<Stream[]>(
                    streamsKeys.list,
                    previousList.map((s) =>
                        s.id === variables.id ? { ...s, ...variables.data } : s
                    )
                );
            }
            return { previousStream, previousList };
        },
        onError(err, variables, context) {
            if (context?.previousStream) {
                queryClient.setQueryData<Stream>(
                    streamsKeys.detail(variables.id),
                    context.previousStream
                );
            }
            if (context?.previousList) {
                queryClient.setQueryData<Stream[]>(streamsKeys.list, context.previousList);
            }
            toast.error(getErrorMessage(err));
        },
        onSettled(_data, _err, variables) {
            queryClient.invalidateQueries({ queryKey: streamsKeys.detail(variables.id) });
            queryClient.invalidateQueries({ queryKey: streamsKeys.list });
        },
        onSuccess: (_data) => {
            toast.success("Stream updated");
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
