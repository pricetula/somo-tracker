/**
 * Streams feature — hooks for creation and grades query.
 */

"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { createStreams } from "@/lib/api/streams";
import { api } from "@/lib/api/client";
import type { CreateStreamsResponse } from "@/lib/api/streams";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const streamKeys = {
    create: ["stream", "create"] as const,
    grades: ["grades"] as const,
};

export interface GradeLevel {
    id: string;
    education_system_id: string;
    local_label: string;
    tier_stage: string;
    sequence_index: number;
}

export interface GradesResponse {
    code: string;
    message: string;
    grades: GradeLevel[];
    errors: Record<string, string[]>;
}

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

export function useGrades() {
    return useQuery<GradesResponse, Error>({
        queryKey: streamKeys.grades,
        queryFn: async () => api.get<GradesResponse>("/api/school/grades"),
    });
}
