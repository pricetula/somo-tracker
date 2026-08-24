/**
 * Timetable API functions (updated for simplified body-ID design).
 *
 * Tracks (Timetable):
 *   POST   /api/v1/timetable                    — create track (+ optional initial blocks)
 *   PUT    /api/v1/timetable                    — update track (ID in body)
 *   DELETE /api/v1/timetable                    — bulk delete tracks (IDs in body)
 *
 * Blocks (Time Structure):
 *   POST   /api/v1/timetable/blocks             — create blocks (track_id in body)
 *   PUT    /api/v1/timetable/blocks             — update blocks (ID in body)
 *   DELETE /api/v1/timetable/blocks             — bulk delete blocks (IDs in body)
 *
 * Allocations (Slot Assignments):
 *   POST   /api/v1/timetable/allocations        — create allocations (block_id in body)
 *   PUT    /api/v1/timetable/allocations        — update allocations (ID in body)
 *   DELETE /api/v1/timetable/allocations        — bulk delete allocations (IDs in body)
 *
 * Combined view:
 *   GET    /api/v1/timetable                    — get structures + slots
 */

import { api } from "./client";

// ─── Types (Tracks) ────────────────────────────────────────────────────────

export interface TimetableTrack {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_year_id: string;
    academic_term_id?: string;
    name: string;
    description?: string;
    is_default: boolean;
    created_at?: string;
    updated_at?: string;
}

export interface CreateTrackPayload {
    name: string;
    description?: string;
    is_default?: boolean;
    academic_year_id?: string;
    academic_term_id?: string;
    initial_blocks?: CreateTimeBlockPayload[];
}

export interface UpdateTrackPayload {
    id: string;
    name?: string;
    description?: string;
    is_default?: boolean;
}

export interface BulkDeletePayload {
    ids: string[];
}

// ─── Types (Structure Blocks) ──────────────────────────────────────────────

export interface TimeBlock {
    id: string;
    track_id: string;
    day_of_week: number;
    period_name: string;
    start_time: string;
    end_time: string;
    is_break: boolean;
    order: number;
    created_at?: string;
    updated_at?: string;
}

export interface CreateTimeBlockPayload {
    track_id: string;
    day_of_week: number;
    period_name: string;
    start_time: string;
    end_time: string;
    is_break?: boolean;
    order?: number;
}

export interface UpdateTimeBlockPayload {
    id: string;
    day_of_week?: number;
    period_name?: string;
    start_time?: string;
    end_time?: string;
    is_break?: boolean;
    order?: number;
}

// ─── Types (Allocations) ──────────────────────────────────────────────────

export interface Allocation {
    id: string;
    tenant_id: string;
    school_id: string;
    block_id: string;
    class_id: string;
    learning_area_id: string;
    teacher_id: string;
    room_identifier?: string | null;
    created_at?: string;
    updated_at?: string;
}

export interface CreateAllocationPayload {
    block_id: string;
    class_id: string;
    learning_area_id: string;
    teacher_id: string;
    room_identifier?: string | null;
}

export interface UpdateAllocationPayload {
    id: string;
    class_id?: string;
    learning_area_id?: string;
    teacher_id?: string;
    room_identifier?: string | null;
}

export interface AllocationFilter {
    block_id?: string;
    class_id?: string;
    teacher_id?: string;
    learning_area_id?: string;
}

// ─── Day helpers ──────────────────────────────────────────────────────────

export const DAY_NAMES: Record<number, string> = {
    1: "Monday",
    2: "Tuesday",
    3: "Wednesday",
    4: "Thursday",
    5: "Friday",
    6: "Saturday",
    7: "Sunday",
};

export const DAY_NAMES_SHORT: Record<number, string> = {
    1: "Mon",
    2: "Tue",
    3: "Wed",
    4: "Thu",
    5: "Fri",
    6: "Sat",
    7: "Sun",
};

export function getDayName(day: number): string {
    return DAY_NAMES[day] ?? `Day ${day}`;
}

export function getDayNameShort(day: number): string {
    return DAY_NAMES_SHORT[day] ?? `D${day}`;
}

// ─── API Functions: Tracks ────────────────────────────────────────────────

/** Create a new timetable track with optional initial blocks. */
export async function createTrack(payload: CreateTrackPayload): Promise<TimetableTrack> {
    return api.post<TimetableTrack>("/api/v1/timetable", payload);
}

/** Update a timetable track (ID passed in body). */
export async function updateTrack(payload: UpdateTrackPayload): Promise<TimetableTrack> {
    return api.put<TimetableTrack>("/api/v1/timetable", payload);
}

/** Bulk delete tracks (IDs passed in body). */
export async function bulkDeleteTracks(
    payload: BulkDeletePayload
): Promise<{ deleted: number; total: number }> {
    return api.delete<{ deleted: number; total: number }>("/api/v1/timetable", payload);
}

// ─── API Functions: Blocks ────────────────────────────────────────────────

/** Create new blocks for a track (track_id in body). */
export async function createBlocks(payload: CreateTimeBlockPayload[]): Promise<TimeBlock[]> {
    return api.post<TimeBlock[]>("/api/v1/timetable/blocks", payload);
}

/** Update a block (ID in body). */
export async function updateBlock(payload: UpdateTimeBlockPayload): Promise<TimeBlock> {
    return api.put<TimeBlock>("/api/v1/timetable/blocks", payload);
}

/** Bulk delete blocks (IDs in body). */
export async function bulkDeleteBlocks(
    payload: BulkDeletePayload
): Promise<{ deleted: number; total: number }> {
    return api.delete<{ deleted: number; total: number }>("/api/v1/timetable/blocks", payload);
}

// ─── API Functions: Allocations ────────────────────────────────────────────

/** Create new allocations for a block (block_id in body). */
export async function createAllocations(payload: CreateAllocationPayload[]): Promise<Allocation[]> {
    return api.post<Allocation[]>("/api/v1/timetable/allocations", payload);
}

/** Update an allocation (ID in body). */
export async function updateAllocation(payload: UpdateAllocationPayload): Promise<Allocation> {
    return api.put<Allocation>("/api/v1/timetable/allocations", payload);
}

/** Bulk delete allocations (IDs in body). */
export async function bulkDeleteAllocations(
    payload: BulkDeletePayload
): Promise<{ deleted: number; total: number }> {
    return api.delete<{ deleted: number; total: number }>("/api/v1/timetable/allocations", payload);
}

// ─── API Functions: Combined View ────────────────────────────────────────

/** Get combined timetable view (blocks + allocations). */
export async function getTimetable(): Promise<{
    blocks: TimeBlock[];
    allocations: Allocation[];
}> {
    return api.get<{
        blocks: TimeBlock[];
        allocations: Allocation[];
    }>("/api/v1/timetable");
}

// ─── Legacy aliases for backward compatibility (if needed) ──────────────

export type Slot = Allocation;
export type CreateSlotPayload = CreateAllocationPayload;
export type UpdateSlotPayload = UpdateAllocationPayload;
export type SlotFilter = AllocationFilter;
export type TimetableAllocation = Allocation;

// Legacy enriched slot type for UI components

export interface EnrichedSlot {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_year_id: string;
    structure_id: string;
    class_id: string;
    learning_area_id?: string | null;
    teacher_id?: string | null;
    room_identifier?: string | null;
    created_at?: string;
    updated_at?: string;
    class_name: string;
    period_name: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
    is_break: boolean;
    learning_area_name?: string | null;
    teacher_name?: string | null;
    session_status?: string | null;
    skip_reason?: string | null;
}
