/**
 * Timetable API functions — matches backend exactly.
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
 * Allocations:
 *   POST   /api/v1/timetable/allocations        — create allocations (block_id in body)
 *   PUT    /api/v1/timetable/allocations        — update allocations (ID in body)
 *   DELETE /api/v1/timetable/allocations        — bulk delete allocations (IDs in body)
 *
 * Combined view:
 *   GET    /api/v1/timetable                    — get blocks + allocations
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

// ─── Types (Blocks) ────────────────────────────────────────────────────────

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

// ─── Types (Allocations) — matches backend Allocation exactly ──────────────

export interface Allocation {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_year_id: string;
    block_id: string;
    class_id: string;
    learning_area_id: string;
    teacher_id: string;
    room_identifier?: string | null;
    created_at?: string;
    updated_at?: string;

    // Joined fields (always populated by GET endpoints)
    class_name: string;
    learning_area_name: string;
    learning_area_code: string;
    teacher_name: string;
    room_name?: string;
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
    tenant_id: string;
    school_id: string;
    academic_year_id?: string;
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
    const result = await api.post<{ track: TimetableTrack; message: string }>(
        "/api/v1/timetable",
        payload
    );
    return result.track;
}

/** Update a timetable track (ID passed in body). */
export async function updateTrack(payload: UpdateTrackPayload): Promise<TimetableTrack> {
    const result = await api.put<{ updated: boolean; track: TimetableTrack }>(
        "/api/v1/timetable",
        payload
    );
    return result.track ?? payload;
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

/** Get combined timetable view (blocks + allocations with joined names). */
export async function getTimetable(
    query?: string,
    teacherId?: string
): Promise<{
    blocks: TimeBlock[];
    allocations: Allocation[];
}> {
    const base = `/api/v1/timetable${query ?? ""}`;
    const teacherQuery = teacherId
        ? `${query ? "&" : "?"}teacher_id=${encodeURIComponent(teacherId)}`
        : "";
    return api.get<{
        blocks: TimeBlock[];
        allocations: Allocation[];
    }>(`${base}${teacherQuery}`);
}

// ─── API Functions: Track List / Single ───────────────────────────────────

/** List all timetable tracks for the active school. */
export async function getTracks(): Promise<{ items: TimetableTrack[]; total: number }> {
    return api.get<{ items: TimetableTrack[]; total: number }>("/api/v1/timetable/tracks");
}

/** Get a single timetable track by ID. */
export async function getTrack(trackId: string): Promise<TimetableTrack> {
    return api.get<TimetableTrack>(`/api/v1/timetable/tracks/${trackId}`);
}

/** Update a track's default status (convenience endpoint). */
export async function setDefaultTrack(trackId: string): Promise<TimetableTrack> {
    const result = await api.put<{ updated: boolean; track: TimetableTrack }>("/api/v1/timetable", {
        id: trackId,
        is_default: true,
    });
    return result.track ?? (await getTrack(trackId));
}

/** Create a track with initial blocks (server replicates to all 7 days). */
export async function createTrackWithBlocks(payload: CreateTrackPayload): Promise<{
    track: TimetableTrack;
    message: string;
}> {
    return api.post<{ track: TimetableTrack; message: string }>("/api/v1/timetable", payload);
}
