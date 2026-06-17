"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { fetchClasses, fetchGrades, generateClasses } from "@/lib/api/classes";
import { getApiErrorMessage } from "@/lib/api/auth";
import type { ClassListParams, GeneratePayload } from "@/features/classes/types";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const classKeys = {
    all: ["classes"] as const,
    current: ["classes", "current"] as const,
    filtered: (params: ClassListParams) => ["classes", "filtered", params] as const,
    grades: ["classes", "grades"] as const,
} as const;

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch all classes for the current school and academic year.
 *
 * Returns an empty array when no classes exist yet (triggering Step 2).
 */
export function useClasses() {
    return useQuery({
        queryKey: classKeys.current,
        queryFn: () => fetchClasses(),
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        retry: 1,
    });
}

/**
 * Fetch classes with optional filter params (grade_ids, search, is_active).
 */
export function useClassList(params: ClassListParams) {
    return useQuery({
        queryKey: classKeys.filtered(params),
        queryFn: () => fetchClasses(params),
        staleTime: 10_000,
        refetchOnWindowFocus: false,
        retry: 1,
    });
}

/**
 * Fetch all grades for the school's education system.
 */
export function useGrades() {
    return useQuery({
        queryKey: classKeys.grades,
        queryFn: fetchGrades,
        staleTime: 300_000, // 5 min — grades don't change often
        refetchOnWindowFocus: false,
        retry: 1,
    });
}

/**
 * Generate (bulk-create) classrooms from stream names.
 *
 * On success:
 *   1. Show a checkmark / toast
 *   2. Invalidate the classes query to refresh the decision tree
 */
export function useGenerateClasses() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: GeneratePayload) => generateClasses(payload),
        onSuccess: async (result) => {
            await queryClient.invalidateQueries({ queryKey: classKeys.current });
            toast.success("Classrooms created!", {
                description: `${result.total_created} classrooms were generated across ${result.streams.length} stream(s).`,
            });
        },
        onError: (err: unknown) => {
            toast.error("Failed to generate classrooms", {
                description: getApiErrorMessage(err),
            });
        },
    });
}
