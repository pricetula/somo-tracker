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

/** Create a new time block. */
export function useCreateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTimeBlockPayload) => createTimeBlock(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
            toast.success("Time block added successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Batch-create time blocks (atomic template application). */
export function useBatchCreateTimeBlocks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BatchCreateTimeBlockPayload) => batchCreateTimeBlocks(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
            toast.success("Template applied successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Replicate one day's schedule to target days. */
export function useReplicateDay() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: ReplicateDayPayload) => replicateDay(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
            toast.success("Schedule replicated successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update an existing time block. */
export function useUpdateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, ...payload }: { id: string } & CreateTimeBlockPayload) =>
            updateTimeBlock(id, payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
            toast.success("Time block updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a time block. */
export function useDeleteTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteTimeBlock(id),
        onSuccess: (data) => {
            void queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
            toast.success(data.message ?? "Time block removed successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
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

/** Fetch enriched slots for the scheduling board. */
export function useEnrichedSlotList(
    academicYearID: string,
    viewBy?: {
        mode: "class" | "teacher" | "room";
        id: string;
    }
) {
    return useQuery({
        queryKey: timetableSlotKeys.enriched({ academicYearID, ...(viewBy ?? {}) }),
        queryFn: () => listEnrichedSlots(academicYearID, viewBy),
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

/** Create a single slot. */
export function useCreateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateSlotPayload) => createSlot(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
            toast.success("Slot assigned successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Batch-create slots. */
export function useBatchCreateSlots() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BatchCreateSlotsPayload) => batchCreateSlots(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
            toast.success("Slots assigned successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a slot. */
export function useUpdateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, ...payload }: { id: string } & UpdateSlotPayload) =>
            updateSlot(id, payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
            toast.success("Slot updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a slot. */
export function useDeleteSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSlot(id),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: timetableSlotKeys.all });
            toast.success("Slot removed");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
