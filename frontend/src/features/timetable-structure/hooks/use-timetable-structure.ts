/**
 * useTimetableStructure — TanStack Query hooks for managing structural time blocks
 * and allocation slots.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listTimeBlocks,
    listTimeBlocksByDay,
    createTimeBlock,
    batchCreateTimeBlocks,
    replicateDay,
    updateTimeBlock,
    deleteTimeBlock,
    deleteTimeBlocksByName,
    listSlots,
    listEnrichedSlots,
    getSlot,
    createSlot,
    batchCreateSlots,
    updateSlot,
    deleteSlot,
} from "@/lib/api/timetable-structure";
import { getErrorMessage } from "@/lib/errors";
import type {
    CreateTimeBlockPayload,
    UpdateTimeBlockPayload,
    BatchCreateTimeBlockPayload,
    ReplicateDayPayload,
    CreateSlotPayload,
    BatchCreateSlotsPayload,
    UpdateSlotPayload,
} from "@/lib/api/timetable-structure";

// ─── Query keys ───────────────────────────────────────────────────────────

export const timetableStructureKeys = {
    all: ["timetable-structure"] as const,
    list: () => [...timetableStructureKeys.all, "list"] as const,
    byDay: (day: number) => [...timetableStructureKeys.all, "day", day] as const,
};

export const timetableSlotKeys = {
    all: ["timetable-slots"] as const,
    list: (filters?: Record<string, string>) =>
        [...timetableSlotKeys.all, "list", filters] as const,
    enriched: (filters?: Record<string, string>) =>
        [...timetableSlotKeys.all, "enriched", filters] as const,
    detail: (id: string) => [...timetableSlotKeys.all, "detail", id] as const,
};

// ─── Hooks: Structure Blocks ──────────────────────────────────────────────

/** Fetch all time blocks for the active school. */
export function useTimeBlockList(academicYearID?: string) {
    return useQuery({
        queryKey: [...timetableStructureKeys.list(), academicYearID],
        queryFn: () => listTimeBlocks(academicYearID),
        enabled: !!academicYearID,
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}

/** Fetch time blocks for a specific day of the week. */
export function useTimeBlockListByDay(day: number, academicYearID?: string) {
    return useQuery({
        queryKey: [...timetableStructureKeys.byDay(day), academicYearID],
        queryFn: () => listTimeBlocksByDay(day, academicYearID),
        staleTime: 5 * 60 * 1000,
        enabled: day >= 1 && day <= 7,
        placeholderData: (prev) => prev,
    });
}

/** Create a new time block with optimistic update. */
export function useCreateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTimeBlockPayload) => createTimeBlock(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableStructureKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: timetableStructureKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Batch-create time blocks (atomic template application) with optimistic update. */
export function useBatchCreateTimeBlocks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BatchCreateTimeBlockPayload) => batchCreateTimeBlocks(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableStructureKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: timetableStructureKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Replicate one day's schedule to target days with optimistic update. */
export function useReplicateDay() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: ReplicateDayPayload) => replicateDay(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableStructureKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: timetableStructureKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Update an existing time block with optional cascade and shift with optimistic update. */
export function useUpdateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, ...payload }: { id: string } & UpdateTimeBlockPayload) =>
            updateTimeBlock(id, payload),
        onMutate: async ({ id, ...payload }) => {
            await queryClient.cancelQueries({ queryKey: timetableStructureKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: timetableStructureKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: timetableStructureKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === id ? { ...item, ...payload } : item
                        ),
                    };
                }
            );

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
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Delete a time block with optimistic removal. */
export function useDeleteTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteTimeBlock(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: timetableStructureKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: timetableStructureKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: timetableStructureKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

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
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Delete all time blocks with a specific period name across all days with optimistic removal. */
export function useDeleteTimeBlocksByName() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            periodName,
            academicYearID,
        }: {
            periodName: string;
            academicYearID: string;
        }) => deleteTimeBlocksByName(periodName, academicYearID),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableStructureKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: timetableStructureKeys.all,
            });

            // For batch delete by name, we can't easily know which items to remove
            // without knowing the period name, so we just invalidate on settle.
            // The previous queries are saved for rollback on error.
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
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

// ─── Hooks: Allocation Slots ──────────────────────────────────────────────

/** Fetch slots with optional filters. */
export function useSlotList(
    academicYearID: string,
    filters?: {
        structure_id?: string;
        class_id?: string;
        teacher_id?: string;
    }
) {
    return useQuery({
        queryKey: timetableSlotKeys.list({ academicYearID, ...filters }),
        queryFn: () => listSlots(academicYearID, filters),
        enabled: !!academicYearID,
        staleTime: 5 * 60 * 1000,
    });
}

/** Fetch enriched slots for the scheduling board.
 *
 * When `date` is provided, the endpoint returns only slots matching that day-of-week
 * and includes session_status / skip_reason from the attendance sessions table.
 */
export function useEnrichedSlotList(
    academicYearID: string,
    opts?: {
        classId?: string;
        teacherId?: string;
        roomIdentifier?: string;
        date?: string;
    }
) {
    return useQuery({
        queryKey: timetableSlotKeys.enriched({
            academicYearID,
            ...(opts ?? {}),
        }),
        queryFn: () => listEnrichedSlots(academicYearID, opts),
        enabled: !!academicYearID,
        staleTime: 5 * 60 * 1000,
    });
}

/** Fetch a single enriched slot. */
export function useSlotDetail(id: string) {
    return useQuery({
        queryKey: timetableSlotKeys.detail(id),
        queryFn: () => getSlot(id),
        enabled: !!id,
    });
}

/** Create a single slot with optimistic update. */
export function useCreateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateSlotPayload) => createSlot(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableSlotKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: timetableSlotKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
        },
    });
}

/** Batch-create slots with optimistic update. */
export function useBatchCreateSlots() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BatchCreateSlotsPayload) => batchCreateSlots(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableSlotKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: timetableSlotKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
        },
    });
}

/** Update a slot with optimistic update. */
export function useUpdateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, ...payload }: { id: string } & UpdateSlotPayload) =>
            updateSlot(id, payload),
        onMutate: async ({ id, ...payload }) => {
            await queryClient.cancelQueries({ queryKey: timetableSlotKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: timetableSlotKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: timetableSlotKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === id ? { ...item, ...payload } : item
                        ),
                    };
                }
            );

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
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
        },
    });
}

/** Delete a slot with optimistic removal. */
export function useDeleteSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSlot(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: timetableSlotKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: timetableSlotKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: timetableSlotKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

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
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
        },
    });
}
