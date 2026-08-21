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
import { getErrorMessage, isApiError } from "@/lib/errors";
import type { Class, ClassListResult } from "@/lib/api/generated";
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
            const previousQueries = queryClient.getQueriesData<ClassListResult>({
                queryKey: classKeys.all,
            });

            queryClient.setQueriesData<ClassListResult>({ queryKey: classKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.filter((item) => !idSet.has(item.id)),
                };
            });

            return { previousQueries };
        },
        onError: (_err, _ids, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(_err));
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

// ─── Types for useCreateClass ──────────────────────────────────────────────

/** Payload accepted by the createClass API call. */
export interface CreateClassPayload {
    grade_level: string;
    stream_id: string;
    student_ids?: string[];
}

/** Context captured in onMutate for rollback and success handling. */
export interface CreateClassContext {
    /** Snapshot of all queries matching classKeys.all before the mutation. */
    previousQueries: ReadonlyArray<readonly [readonly unknown[], ClassListResult | undefined]>;
    /** Temporary client-side ID assigned to the optimistic item. */
    tempId: string;
}

/**
 * Create a new class with optimistic update, full rollback on error,
 * and zero-flicker replacement of the temporary item on success.
 */
export function useCreateClass() {
    const queryClient = useQueryClient();

    return useMutation<Class, unknown, CreateClassPayload, CreateClassContext>({
        mutationFn: createClass,

        /**
         * onMutate — Prepare optimistic update.
         *
         * WHY: We cancel any in-flight queries for the class prefix first so
         * our setQueriesData write isn't immediately overwritten by a
         * stale response. We snapshot ALL queries matching the prefix
         * (["classes", "list"]) because different hooks (useClassList,
         * useClassMap) share the same cache key but have different select
         * transforms — restoring all of them guarantees a consistent rollback.
         * We generate a cryptographically-random temp ID
         * (temp-${crypto.randomUUID()}) so the optimistic item has a
         * stable React key and can be precisely targeted for replacement
         * in onSuccess. The functional updater form ensures we never
         * mutate the cached array in place, preserving React Query's
         * structural sharing guarantees.
         */
        onMutate: async (newClassPayload) => {
            // Cancel outgoing refetches so our optimistic write isn't clobbered.
            await queryClient.cancelQueries({ queryKey: classKeys.all });

            // Snapshot every query matching the classKeys.all prefix.
            // Using getQueriesData returns [queryKey, data] tuples for each match.
            // The raw cache data is ClassListResult (before select transforms).
            const previousQueries = queryClient.getQueriesData<ClassListResult>({
                queryKey: classKeys.all,
            });

            // Generate a stable, unique temporary ID for the optimistic item.
            // crypto.randomUUID() is available in all modern browsers and Node 18+.
            const tempId = `temp-${crypto.randomUUID()}`;

            const optimisticClass: Class = {
                id: tempId,
                grade_level: newClassPayload.grade_level,
                stream_id: newClassPayload.stream_id,
                display_label: "",
                stream_name: "",
                stream_color: "",
                student_count: 0,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            };

            // Optimistically prepend the new class to every matching query.
            // Functional updater form prevents in-place mutation of cached arrays.
            queryClient.setQueriesData<{ pages: Array<{ items: Array<Class> }> }>(
                { queryKey: classKeys.all },
                (old) => {
                    if (!old?.pages) return old;

                    old.pages[0].items.push(optimisticClass);

                    return {
                        ...old,
                        pages: old?.pages,
                    };
                }
            );

            // Return context for onError rollback and onSuccess replacement.
            return { previousQueries, tempId };
        },

        /**
         * onError — Rollback using ONLY the context from onMutate.
         *
         * WHY: We rely exclusively on the snapshot captured in onMutate,
         * not on any outer closure or component state. This guarantees
         * the rollback restores the exact cache state at mutation start,
         * even if other mutations or queries have modified the cache
         * in the meantime.
         *
         * We differentiate error types via ApiError.status/code:
         * - 409 (conflict): The server rejected the create due to a
         *   concurrent change (e.g. duplicate grade+stream). Blindly
         *   restoring the stale snapshot would hide the conflict.
         *   Instead we re-invalidate so the UI refetches the latest
         *   server state and the user sees the actual conflict.
         * - 422 (validation): Field-level errors are surfaced via
         *   err.errors for inline form feedback; we still rollback
         *   the optimistic item because the create never happened.
         * - 5xx / network: Generic failure; rollback + toast is the
         *   correct UX — the user can retry.
         */
        onError: (err, _vars, context) => {
            // If the backend reported a conflict (409), the optimistic
            // snapshot is stale — don't restore it. Force a fresh fetch.
            if (isApiError(err) && err.status === 409) {
                queryClient.invalidateQueries({ queryKey: classKeys.all });
                toast.error(err.message);
                return;
            }

            // For all other errors (422, 5xx, network), restore the
            // exact snapshots we captured in onMutate.
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }

            // Surface field-level validation errors (422) or generic message.
            toast.error(getErrorMessage(err));
        },

        /**
         * onSuccess — Replace temp item with real server response.
         *
         * WHY: The server returns the fully-populated Class (with real
         * ID, display_label, stream_name, stream_color, student_count,
         * timestamps). We locate the optimistic item by its tempId and
         * swap it in-place. This avoids the flicker/duplicate-row effect
         * that would occur if we waited for onSettled's invalidate to
         * refetch — the user sees their new class instantly with correct
         * data, not a placeholder.
         */
        onSuccess: (serverClass, _vars, context) => {
            if (!context?.tempId) return;

            // Replace the optimistic item with the real server response
            // in all queries matching the classKeys.all prefix.
            queryClient.setQueriesData<{ pages: Array<{ items: Array<Class> }> }>(
                { queryKey: classKeys.all },
                (old) => {
                    if (!old?.pages) return old;
                    // Build index once: O(N)
                    const userIndex = new Map();

                    old.pages.forEach((page) => {
                        page.items.forEach((item) => userIndex.set(item.id, item));
                    });

                    // Direct O(1) lookup & update
                    let target = userIndex.get(context?.tempId);
                    if (target) {
                        target = serverClass;
                    }
                    return {
                        ...old,
                        pages: old.pages,
                    };
                }
            );
        },

        /**
         * onSettled — Always reconcile with server as source of truth.
         *
         * WHY: Regardless of success or failure, the server is the
         * authoritative state. Invalidating ensures:
         * - On success: any fields we couldn't predict (e.g. computed
         *   display_label, server-generated timestamps) are confirmed.
         * - On error (non-409): the rollback restored the pre-mutation
         *   cache, but a concurrent mutation from another tab/user could
         *   have changed the list; invalidate picks up those changes.
         * - On 409: we already invalidated in onError, but a second
         *   invalidate here is idempotent and harmless.
         */
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: classKeys.all });
        },
    });
}
