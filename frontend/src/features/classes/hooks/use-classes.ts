/**
 * useClassList — TanStack Query hook for fetching classes.
 *
 * The backend defaults to the current academic year, so no params needed.
 * Stable query key + generous staleTime ensures N simultaneously mounted
 * <ClassCombobox /> instances produce exactly ONE network request.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { listClasses, bulkDeleteClasses, createClass } from "@/lib/api/classes";
import { STALE_TIMES } from "@/lib/query-config";
import { getErrorMessage } from "@/lib/errors";
import type { Class } from "../types";
import type { ClassOption } from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const classKeys = {
    all: ["classes"] as const,
    list: () => [...classKeys.all, "list"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch all classes as an array (combobox-friendly select transform).
 */
export function useClassList() {
    return useQuery({
        queryKey: classKeys.list(),
        queryFn: () => listClasses({ limit: 500 }),
        staleTime: STALE_TIMES.REFERENCE_DATA,
        placeholderData: (prev) => prev,
        select: (data): { items: ClassOption[] } => ({
            items: data.items.map((c) => ({
                value: c.id,
                label: c.display_label,
            })),
        }),
    });
}

/** Bulk delete classes with optimistic removal. */
export function useDeleteClasses() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (ids: string[]) => bulkDeleteClasses(ids),
        onMutate: async (ids) => {
            const idSet = new Set(ids);
            await queryClient.cancelQueries({ queryKey: classKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: Class[] }>({
                queryKey: classKeys.all,
            });

            queryClient.setQueriesData<{ items: Class[] }>({ queryKey: classKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.filter((item) => !idSet.has(item.id)),
                };
            });

            return { previousQueries };
        },
        onError: (err, _ids, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: classKeys.all });
        },
    });
}

/**
 * Fetch all classes as a Record<id, Class> for O(1) lookups by ID.
 *
 * Shares the same cache entry as useClassList — no extra network request.
 * Useful for lookup tables, detail panels, and cross-reference from other
 * entities that reference a class by id.
 *
 * @example
 *   const { data: classMap } = useClassMap();
 *   const className = classMap?.[classId]?.display_label;
 */
export function useClassMap() {
    return useQuery({
        queryKey: classKeys.list(),
        queryFn: () => listClasses({ limit: 500 }),
        staleTime: STALE_TIMES.REFERENCE_DATA,
        placeholderData: (prev) => prev,
        select: (data): Record<string, Class> =>
            data?.items?.reduce?.(
                (acc, item) => {
                    acc[item.id] = item;
                    return acc;
                },
                {} as Record<string, Class>
            ) ?? {},
    });
}

/**
 * Create a new class with optimistic update.
 */
export function useCreateClass() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: { grade_level: string; stream_id: string; student_ids?: string[] }) =>
            createClass(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: classKeys.all });
            toast.success("Class created successfully.");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });
}
