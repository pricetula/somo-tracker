/**
 * Streams feature — hooks for creation.
 */

"use client";

import { useMutation } from "@tanstack/react-query";
import { createStreams } from "@/lib/api/streams";
import type { CreateStreamsResponse } from "@/lib/api/streams";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const streamKeys = {
    create: ["stream", "create"] as const,
};

export function useCreateStreams() {
    return useMutation<CreateStreamsResponse, Error, string[]>({
        mutationKey: streamKeys.create,
        mutationFn: (names) => createStreams(names),
        onSuccess: (data) => {
            toast.success(data.message ?? "Streams created successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
