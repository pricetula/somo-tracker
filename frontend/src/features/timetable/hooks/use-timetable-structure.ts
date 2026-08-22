"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listTimeBlocks,
    listTimeBlocksByDay,
    createTimeBlock,
    batchCreateTimeBlocks,
    replicateDay,
    updateTimeBlock,
    deleteTimeBlock,
    deleteDayBlocks,
    deleteTimeBlocksByName,
    type CreateTimeBlockPayload,
    type BatchCreateTimeBlockPayload,
    type ReplicateDayPayload,
    type UpdateTimeBlockPayload,
} from "@/lib/api/timetable-structure";

/** Query key factory for timetable structure */
export const timetableStructureKeys = {
    all: ["timetable", "structure"] as const,
    list: (academicYearId: string) =>
        [...timetableStructureKeys.all, "list", academicYearId] as const,
    day: (academicYearId: string, day: number) =>
        [...timetableStructureKeys.all, "day", academicYearId, day] as const,
};

/** Fetch all time blocks for an academic year */
export function useTimeBlocks(academicYearId: string) {
    return useQuery({
        queryKey: timetableStructureKeys.list(academicYearId),
        queryFn: () => listTimeBlocks(academicYearId),
        enabled: !!academicYearId,
    });
}

/** Fetch time blocks for a specific day */
export function useTimeBlocksByDay(academicYearId: string, day: number) {
    return useQuery({
        queryKey: timetableStructureKeys.day(academicYearId, day),
        queryFn: () => listTimeBlocksByDay(day, academicYearId),
        enabled: !!academicYearId && day >= 1 && day <= 7,
    });
}

/** Create a single time block */
export function useCreateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTimeBlockPayload) => createTimeBlock(payload),
        onSuccess: (_data, _variables, _context) => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Batch create time blocks */
export function useBatchCreateTimeBlocks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BatchCreateTimeBlockPayload) => batchCreateTimeBlocks(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Replicate a day's schedule to other days */
export function useReplicateDay() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: ReplicateDayPayload) => replicateDay(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Update a time block */
export function useUpdateTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateTimeBlockPayload }) =>
            updateTimeBlock(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Delete a single time block */
export function useDeleteTimeBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteTimeBlock(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Delete all time blocks for a day */
export function useDeleteDayBlocks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (day: number) => deleteDayBlocks(day),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}

/** Delete all time blocks with a given period name */
export function useDeleteTimeBlocksByName() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (periodName: string) => deleteTimeBlocksByName(periodName),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableStructureKeys.all });
        },
    });
}
