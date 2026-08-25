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
    type CreateTrackPayload,
    type AllocationFilter,
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.tracks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.tracks.all });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.tracks.all });
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: timetableKeys.blocks.all });
            queryClient.invalidateQueries({ queryKey: ["timetable", "allocations"] });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["timetable", "allocations"] });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["timetable", "allocations"] });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["timetable", "allocations"] });
        },
    });
}

/**
 * List allocations with filters.
 */
export function useAllocations(filters: {
    block_id?: string;
    class_id?: string;
    teacher_id?: string;
    learning_area_id?: string;
}) {
    // Note: Backend GET /api/v1/timetable returns full view; use getTimetable instead
    // This hook kept for compatibility if a dedicated list endpoint is added
    return useQuery({
        queryKey: ["timetable", "allocations", "list", filters],
        queryFn: async () => {
            const data = await getTimetable();
            return data.allocations.filter((a: Allocation) => {
                if (filters.block_id && a.block_id !== filters.block_id) return false;
                if (filters.class_id && a.class_id !== filters.class_id) return false;
                if (filters.teacher_id && a.teacher_id !== filters.teacher_id) return false;
                if (filters.learning_area_id && a.learning_area_id !== filters.learning_area_id)
                    return false;
                return true;
            });
        },
        enabled: true,
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
    allocationByDay: Record<number, Allocation>; // Keyed by day_of_week (e.g., 1: Allocation)
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
            allocationByDay: Record<number, Allocation>; // Direct object mapping instead of array
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
            });
        }
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
 * Get combined timetable view (blocks + allocations).
 */
export function useTimetableView() {
    return useQuery({
        queryKey: ["timetable", "combined"],
        queryFn: getTimetable,
        select: timetableViewSelect,
    });
}
