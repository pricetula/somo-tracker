/**
 * React Query hooks for the Timetable feature (updated for body-ID design).
 */
"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createTrack,
    updateTrack,
    bulkDeleteTracks,
    createBlocks,
    updateBlock,
    bulkDeleteBlocks,
    createAllocations,
    updateAllocation,
    bulkDeleteAllocations,
    getTimetable,
    getTrack,
    getTracks,
    type AllocationFilter,
    type CreateTrackPayload,
    type Allocation,
    type TimeBlock,
    DAY_NAMES,
    DAY_NAMES_SHORT,
} from "@/lib/api/timetable";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const timetableKeys = {
    all: ["timetable"] as const,
    tracks: {
        all: ["timetable", "tracks"] as const,
    },
    blocks: {
        all: ["timetable", "blocks"] as const,
    },
    allocations: {
        all: ["timetable", "allocations"] as const,
        list: (filters: AllocationFilter) => ["timetable", "allocations", "list", filters] as const,
    },
    combined: {
        all: ["timetable", "combined"] as const,
    },
};

// ─── Track Hooks ──────────────────────────────────────────────────────────

/**
 * Create a new timetable track (with optional initial blocks).
 */
export function useCreateTrack() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTrackPayload) => createTrack(payload),
        onMutate: async (_payload) => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _payload, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.tracks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

/**
 * Update a timetable track (ID in body).
 */
export function useUpdateTrack() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (
            payload: { id: string } & { name?: string; description?: string; is_default?: boolean }
        ) => updateTrack(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.tracks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

/**
 * Bulk delete tracks (IDs in body).
 */
export function useBulkDeleteTracks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (ids: string[]) => bulkDeleteTracks({ ids }),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.tracks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

// ─── Block Hooks ──────────────────────────────────────────────────────────

/**
 * Create new blocks for a track (track_id in body).
 */
export function useCreateBlocks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (
            payload: {
                track_id: string;
                day_of_week: number;
                period_name: string;
                start_time: string;
                end_time: string;
                is_break?: boolean;
                order?: number;
            }[]
        ) => createBlocks(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

/**
 * Update a block (ID in body).
 */
export function useUpdateBlock() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: {
            id: string;
            day_of_week?: number;
            period_name?: string;
            start_time?: string;
            end_time?: string;
            is_break?: boolean;
            order?: number;
        }) => updateBlock(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

/**
 * Bulk delete blocks (IDs in body).
 */
export function useBulkDeleteBlocks() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (ids: string[]) => bulkDeleteBlocks({ ids }),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.allocations.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

// ─── Allocation Hooks ─────────────────────────────────────────────────────

/**
 * Create new allocations for a block (block_id in body).
 */
export function useCreateAllocations() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (
            payload: {
                block_id: string;
                class_id: string;
                learning_area_id: string;
                teacher_id: string;
                room_identifier?: string | null;
            }[]
        ) => createAllocations(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.allocations.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

/**
 * Update an allocation (ID in body).
 */
export function useUpdateAllocation() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: {
            id: string;
            class_id?: string;
            learning_area_id?: string;
            teacher_id?: string;
            room_identifier?: string | null;
        }) => updateAllocation(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.allocations.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

/**
 * Bulk delete allocations (IDs in body).
 */
export function useBulkDeleteAllocations() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (ids: string[]) => bulkDeleteAllocations({ ids }),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: timetableKeys.combined.all });
            const previous = queryClient.getQueryData(timetableKeys.combined.all);
            return { previous };
        },
        onError: (_err, _vars, context) => {
            if (context?.previous) {
                queryClient.setQueryData(timetableKeys.combined.all, context.previous);
            }
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.allocations.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.combined.all });
        },
    });
}

// ─── Combined View Hook ──────────────────────────────────────────────────
export interface TimetableDay {
    day_of_week: number;
    name: string;
    shortName: string;
}

export interface TimetableRow {
    period_name: string;
    start_time: string;
    end_time: string;
    order: number;
    is_break: boolean;
    blockIdByDay: Record<number, string>; // block id per day (correct for links/allocations)
    allocationByDay: Record<number, Allocation>;
}

export interface TimetableViewResult {
    days: TimetableDay[];
    rows: TimetableRow[];
}

export function timetableViewSelect({
    blocks,
    allocations,
}: {
    blocks: TimeBlock[];
    allocations: Allocation[];
}): TimetableViewResult {
    const daysMap = new Map<number, { day_of_week: number; name: string; shortName: string }>();
    const rowMap = new Map<
        string,
        {
            period_name: string;
            start_time: string;
            end_time: string;
            order: number;
            is_break: boolean;
            allocationByDay: Record<number, Allocation>;
            blockIdByDay: Record<number, string>;
        }
    >();

    // Fast lookup: block_id -> { rowKey, day_of_week }
    const blockMetaMap = new Map<string, { rowKey: string; day_of_week: number }>();

    // 1. Process blocks once: O(N)
    blocks.forEach((block) => {
        if (!daysMap.has(block.day_of_week)) {
            daysMap.set(block.day_of_week, {
                day_of_week: block.day_of_week,
                name: DAY_NAMES[block.day_of_week] || "Unknown",
                shortName: DAY_NAMES_SHORT[block.day_of_week] || "Unk",
            });
        }

        const rowKey = `${block.start_time}-${block.end_time}-${block.period_name}`;
        blockMetaMap.set(block.id, { rowKey, day_of_week: block.day_of_week });

        if (!rowMap.has(rowKey)) {
            rowMap.set(rowKey, {
                period_name: block.period_name,
                start_time: block.start_time,
                end_time: block.end_time,
                order: block.order,
                is_break: block.is_break,
                allocationByDay: {},
                blockIdByDay: {},
            });
        }
        const row = rowMap.get(rowKey)!;
        row.blockIdByDay[block.day_of_week] = block.id;
    });

    // 2. Process allocations: O(M)
    allocations.forEach((allocation) => {
        const meta = blockMetaMap.get(allocation.block_id);
        if (!meta) return;

        const row = rowMap.get(meta.rowKey);
        if (row) {
            // Directly assign since it's 1-to-1 per class view
            row.allocationByDay[meta.day_of_week] = allocation;
        }
    });

    return {
        days: Array.from(daysMap.values()).sort((a, b) => a.day_of_week - b.day_of_week),
        rows: Array.from(rowMap.values()).sort((a, b) => a.order - b.order),
    };
}

/**
 * Get a single timetable track by ID.
 */
export function useTrack(trackId: string) {
    return useQuery({
        queryKey: ["timetable", "tracks", trackId],
        queryFn: () => getTrack(trackId),
        enabled: !!trackId,
    });
}

/**
 * List all timetable tracks for the active school.
 */
export function useTracks() {
    return useQuery({
        queryKey: timetableKeys.tracks.all,
        queryFn: getTracks,
    });
}

/**
 * Get combined timetable view (blocks + allocations).
 */
export function useTimetableView(trackId?: string) {
    return useQuery({
        queryKey: ["timetable", "combined", trackId ?? "all"],
        queryFn: async () => {
            const params = new URLSearchParams();
            if (trackId) params.set("track_id", trackId);
            const query = params.toString() ? `?${params.toString()}` : "";
            return getTimetable(query);
        },
        select: ({ blocks, allocations }: { blocks: TimeBlock[]; allocations: Allocation[] }) =>
            timetableViewSelect({ blocks, allocations }),
    });
}
