/**
 * Timetable Structure API functions.
 *
 * Endpoints:
 *   GET    /api/v1/timetable/structure              — list all time blocks
 *   GET    /api/v1/timetable/structure/day/:day      — list blocks for a day
 *   POST   /api/v1/timetable/structure               — create a single time block
 *   POST   /api/v1/timetable/structure/batch         — batch-create time blocks (atomic)
 *   POST   /api/v1/timetable/structure/replicate     — replicate one day's schedule to others
 *   PUT    /api/v1/timetable/structure/:id           — update a time block
 *   DELETE /api/v1/timetable/structure/:id           — delete a time block
 *   DELETE /api/v1/timetable/structure/day/:day      — delete all blocks for a day
 *
 * Timetable Slots (Allocation) endpoints:
 *   GET    /api/v1/timetable/slots                   — list slots with optional filters
 *   GET    /api/v1/timetable/slots/enriched          — list slots with joined data
 *   POST   /api/v1/timetable/slots                   — create a single slot
 *   POST   /api/v1/timetable/slots/batch             — batch-create slots
 *   PUT    /api/v1/timetable/slots/:id               — update a slot
 *   DELETE /api/v1/timetable/slots/:id               — delete a slot
 */

import { api } from "./client";

// ─── Types (Structure Blocks) ──────────────────────────────────────────────

export interface TimeBlock {
    id: string;
    day_of_week: number;
    period_name: string;
    start_time: string;
    end_time: string;
    is_break: boolean;
    academic_year_id?: string;
}

export interface TimeBlockListResult {
    items: TimeBlock[];
    total: number;
}

export interface CreateTimeBlockPayload {
    day_of_week: number;
    period_name: string;
    start_time: string;
    end_time: string;
    is_break: boolean;
    academic_year_id: string;
}

export interface UpdateTimeBlockPayload extends CreateTimeBlockPayload {
    /**
     * Propagation mode:
     *   ""         — update only this block (default)
     *   "all_days" — update all blocks with the same period_name on all days
     */
    propagate?: string;
    /** Shift subsequent blocks on the same day by the time delta */
    shift_following?: boolean;
}

export interface BatchCreateTimeBlockPayload {
    blocks: CreateTimeBlockPayload[];
}

export interface ReplicateDayPayload {
    source_day: number;
    target_days: number[];
}

export interface DeleteResult {
    deleted: boolean;
    linked_lessons?: number;
    message?: string;
}

// ─── Types (Allocation Slots) ──────────────────────────────────────────────

export interface TimetableSlot {
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
}

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
}

export interface SlotListResult {
    items: TimetableSlot[];
    total: number;
}

export interface EnrichedSlotListResult {
    items: EnrichedSlot[];
    total: number;
}

export interface CreateSlotPayload {
    academic_year_id: string;
    structure_id: string;
    class_id: string;
    learning_area_id?: string | null;
    teacher_id?: string | null;
    room_identifier?: string | null;
}

export interface BatchCreateSlotsPayload {
    slots: CreateSlotPayload[];
}

export interface UpdateSlotPayload {
    learning_area_id?: string | null;
    teacher_id?: string | null;
    room_identifier?: string | null;
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

// ─── API Functions: Structure Blocks ──────────────────────────────────────

/** List all time blocks for the active school and academic year. */
export async function listTimeBlocks(academicYearID?: string): Promise<TimeBlockListResult> {
    const params = academicYearID ? `?academic_year_id=${encodeURIComponent(academicYearID)}` : "";
    return api.get<TimeBlockListResult>(`/api/v1/timetable/structure${params}`);
}

/** List time blocks for a specific day (1=Monday, 7=Sunday). */
export async function listTimeBlocksByDay(
    day: number,
    academicYearID?: string
): Promise<TimeBlockListResult> {
    const params = academicYearID ? `?academic_year_id=${encodeURIComponent(academicYearID)}` : "";
    return api.get<TimeBlockListResult>(`/api/v1/timetable/structure/day/${day}${params}`);
}

/** Create a new time block. */
export async function createTimeBlock(payload: CreateTimeBlockPayload): Promise<TimeBlock> {
    return api.post<TimeBlock>("/api/v1/timetable/structure", payload);
}

/** Batch-create time blocks (atomic — all or nothing). */
export async function batchCreateTimeBlocks(
    payload: BatchCreateTimeBlockPayload
): Promise<TimeBlockListResult> {
    return api.post<TimeBlockListResult>("/api/v1/timetable/structure/batch", payload);
}

/** Replicate one day's schedule to target days (mass replication). */
export async function replicateDay(payload: ReplicateDayPayload): Promise<TimeBlockListResult> {
    return api.post<TimeBlockListResult>("/api/v1/timetable/structure/replicate", payload);
}

/** Update an existing time block with optional cascade and shift. */
export async function updateTimeBlock(
    id: string,
    payload: UpdateTimeBlockPayload
): Promise<TimeBlockListResult> {
    return api.put<TimeBlockListResult>(`/api/v1/timetable/structure/${id}`, payload);
}

/** Delete a time block by ID. */
export async function deleteTimeBlock(id: string): Promise<DeleteResult> {
    return api.delete<DeleteResult>(`/api/v1/timetable/structure`, { id });
}

/** Delete all time blocks for a specific day. */
export async function deleteDayBlocks(day: number, academicYearID?: string): Promise<void> {
    return api.delete<void>(`/api/v1/timetable/structure/day`, {
        day,
        academic_year_id: academicYearID,
    });
}

/** Delete all time blocks with a given period name across all days. */
export async function deleteTimeBlocksByName(
    periodName: string,
    academicYearID: string
): Promise<DeleteResult> {
    return api.delete<DeleteResult>(`/api/v1/timetable/structure/by-name`, {
        academic_year_id: academicYearID,
        period_name: periodName,
    });
}

// ─── API Functions: Allocation Slots ──────────────────────────────────────

/** List slots with optional filters. */
export async function listSlots(
    academicYearID: string,
    filters?: {
        structure_id?: string;
        class_id?: string;
        teacher_id?: string;
        room_identifier?: string;
    }
): Promise<SlotListResult> {
    const params = new URLSearchParams({ academic_year_id: academicYearID });
    if (filters?.structure_id) params.set("structure_id", filters.structure_id);
    if (filters?.class_id) params.set("class_id", filters.class_id);
    if (filters?.teacher_id) params.set("teacher_id", filters.teacher_id);
    if (filters?.room_identifier) params.set("room_identifier", filters.room_identifier);
    return api.get<SlotListResult>(`/api/v1/timetable/slots?${params.toString()}`);
}

/** List enriched slots with joined data for the scheduling board. */
export async function listEnrichedSlots(
    academicYearID: string,
    viewBy?: {
        mode: "class" | "teacher" | "room";
        id: string;
    }
): Promise<EnrichedSlotListResult> {
    const params = new URLSearchParams({ academic_year_id: academicYearID });
    if (viewBy) {
        params.set("view_by", viewBy.mode);
        if (viewBy.mode === "class") params.set("class_id", viewBy.id);
        else if (viewBy.mode === "teacher") params.set("teacher_id", viewBy.id);
        else if (viewBy.mode === "room") params.set("room_identifier", viewBy.id);
    }
    return api.get<EnrichedSlotListResult>(`/api/v1/timetable/slots/enriched?${params.toString()}`);
}

/** Get a single enriched slot by ID. */
export async function getSlot(id: string): Promise<EnrichedSlot> {
    return api.get<EnrichedSlot>(`/api/v1/timetable/slots/${id}`);
}

/** Create a single slot assignment. */
export async function createSlot(payload: CreateSlotPayload): Promise<TimetableSlot> {
    return api.post<TimetableSlot>("/api/v1/timetable/slots", payload);
}

/** Batch-create slots (atomic). */
export async function batchCreateSlots(payload: BatchCreateSlotsPayload): Promise<SlotListResult> {
    return api.post<SlotListResult>("/api/v1/timetable/slots/batch", payload);
}

/** Update a slot assignment. */
export async function updateSlot(id: string, payload: UpdateSlotPayload): Promise<TimetableSlot> {
    return api.put<TimetableSlot>(`/api/v1/timetable/slots/${id}`, payload);
}

/** Delete a slot by ID. */
export async function deleteSlot(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/timetable/slots`, { id });
}
