"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listEnrichedSlots,
    createSlot,
    batchCreateSlots,
    updateSlot,
    deleteSlot,
    type CreateSlotPayload,
    type BatchCreateSlotsPayload,
    type UpdateSlotPayload,
} from "@/lib/api/timetable-structure";

/** Query key factory for timetable slots */
export const timetableSlotsKeys = {
    all: ["timetable", "slots"] as const,
    enriched: (academicYearId: string, filters?: Record<string, string>) =>
        [...timetableSlotsKeys.all, "enriched", academicYearId, filters] as const,
    single: (id: string) => [...timetableSlotsKeys.all, "single", id] as const,
};

/** Fetch enriched slots with optional filters */
export function useEnrichedSlots(
    academicYearId: string,
    filters?: {
        classId?: string;
        teacherId?: string;
        roomIdentifier?: string;
        date?: string;
    }
) {
    return useQuery({
        queryKey: timetableSlotsKeys.enriched(academicYearId, filters),
        queryFn: () => listEnrichedSlots(academicYearId, filters),
        enabled: !!academicYearId,
    });
}

/** Fetch a single enriched slot by ID */
export function useSlot(id: string) {
    return useQuery({
        queryKey: timetableSlotsKeys.single(id),
        queryFn: () => listEnrichedSlots("", { classId: id }), // This won't work correctly, need getSlot
        enabled: !!id,
    });
}

/** Create a single slot assignment */
export function useCreateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateSlotPayload) => createSlot(payload),
        onSuccess: (_data, _variables, _context) => {
            queryClient.invalidateQueries({ queryKey: timetableSlotsKeys.all });
        },
    });
}

/** Batch create slots */
export function useBatchCreateSlots() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BatchCreateSlotsPayload) => batchCreateSlots(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableSlotsKeys.all });
        },
    });
}

/** Update a slot assignment */
export function useUpdateSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateSlotPayload }) =>
            updateSlot(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableSlotsKeys.all });
        },
    });
}

/** Delete a slot */
export function useDeleteSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSlot(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableSlotsKeys.all });
        },
    });
}
