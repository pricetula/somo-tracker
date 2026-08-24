/**
 * React Query hooks for the Timetable feature.
 */
"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listTimeBlocks,
    getTimeBlock,
    createTimeBlock,
    updateTimeBlock,
    deleteTimeBlock,
    listSlots,
    createSlot,
    batchCreateSlots,
    updateSlot,
    deleteSlot,
    getTimetable,
    type TimeBlock,
    type CreateTimeBlockPayload,
    type UpdateTimeBlockPayload,
    type TimetableSlot,
    type SlotFilter,
    type CreateSlotPayload,
    type BatchCreateSlotsPayload,
    type UpdateSlotPayload,
} from "@/lib/api/timetable-structure";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const timetableKeys = {
    all: ["timetable"] as const,
    structures: {
        all: ["timetable", "structures"] as const,
        list: (academicYearID: string) =>
            ["timetable", "structures", "list", academicYearID] as const,
        detail: (id: string) => ["timetable", "structures", id] as const,
    },
    slots: {
        all: ["timetable", "slots"] as const,
        list: (filters: SlotFilter) => ["timetable", "slots", "list", filters] as const,
    },
    combined: {
        all: ["timetable", "combined"] as const,
        view: (academicYearID: string) => ["timetable", "combined", academicYearID] as const,
    },
};

// ─── Structure (TimeBlock) Hooks ──────────────────────────────────────────

/**
 * List all time blocks for an academic year.
 */
export function useTimeBlocks(academicYearID: string) {
    return useQuery({
        queryKey: timetableKeys.structures.list(academicYearID),
        queryFn: () => listTimeBlocks(academicYearID),
        enabled: !!academicYearID,
    });
}

/**
 * Get a single time block by ID.
 */
export function useTimeBlock(id: string) {
    return useQuery({
        queryKey: timetableKeys.structures.detail(id),
        queryFn: () => getTimeBlock(id),
        enabled: !!id,
    });
}

/**
 * Create a new time block.
 */
export function useCreateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTimeBlockPayload) => createTimeBlock(payload),
        onSuccess: (_data, variables) => {
            // Invalidate the list for the relevant academic year
            if (variables.academic_year_id) {
                queryClient.invalidateQueries({
                    queryKey: timetableKeys.structures.list(variables.academic_year_id),
                });
            }
            queryClient.invalidateQueries({
                queryKey: timetableKeys.structures.all,
            });
        },
    });
}

/**
 * Update an existing time block.
 */
export function useUpdateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateTimeBlockPayload }) =>
            updateTimeBlock(id, payload),
        onSuccess: (_data, variables) => {
            queryClient.invalidateQueries({
                queryKey: timetableKeys.structures.list(variables.payload.academic_year_id),
            });
            queryClient.invalidateQueries({
                queryKey: timetableKeys.structures.detail(variables.id),
            });
        },
    });
}

/**
 * Delete a time block.
 */
export function useDeleteTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteTimeBlock(id),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: timetableKeys.structures.all,
            });
        },
    });
}

// ─── Slot Hooks ───────────────────────────────────────────────────────────

/**
 * List slots with filters.
 */
export function useSlots(filters: SlotFilter) {
    return useQuery({
        queryKey: timetableKeys.slots.list(filters),
        queryFn: () => listSlots(filters),
        enabled: !!filters.academic_year_id,
    });
}

/**
 * Create a single slot.
 */
export function useCreateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateSlotPayload) => createSlot(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: timetableKeys.slots.all,
            });
        },
    });
}

/**
 * Batch create slots.
 */
export function useBatchCreateSlots() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            payload,
            academicYearID,
        }: {
            payload: BatchCreateSlotsPayload;
            academicYearID: string;
        }) => batchCreateSlots(payload, academicYearID),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: timetableKeys.slots.all,
            });
        },
    });
}

/**
 * Update a slot.
 */
export function useUpdateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateSlotPayload }) =>
            updateSlot(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: timetableKeys.slots.all,
            });
        },
    });
}

/**
 * Delete a slot.
 */
export function useDeleteSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSlot(id),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: timetableKeys.slots.all,
            });
        },
    });
}

// ─── Combined View Hook ───────────────────────────────────────────────────

/**
 * Get combined timetable view (structures + slots) for an academic year.
 */
export function useTimetableView(academicYearID: string) {
    return useQuery({
        queryKey: timetableKeys.combined.view(academicYearID),
        queryFn: () => getTimetable(academicYearID),
        enabled: !!academicYearID,
    });
}
