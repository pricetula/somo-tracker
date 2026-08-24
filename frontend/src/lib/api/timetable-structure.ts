/**
 * Timetable Structure API functions.
 *
 * Endpoints:
 *   GET    /api/v1/timetable/structure?academic_year_id=   — list all time blocks
 *   GET    /api/v1/timetable/structure/:id               — get a single time block
 *   POST   /api/v1/timetable/structure                   — create a single time block
 *   PUT    /api/v1/timetable/structure/:id               — update a time block
 *   DELETE /api/v1/timetable/structure/:id               — delete a time block
 *
 * Timetable Slots (Allocation) endpoints:
 *   GET    /api/v1/timetable/slots                       — list slots with optional filters
 *   POST   /api/v1/timetable/slots                       — create a single slot
 *   POST   /api/v1/timetable/slots/batch                 — batch-create slots
 *   PUT    /api/v1/timetable/slots/:id                   — update a slot
 *   DELETE /api/v1/timetable/slots/:id                   — delete a slot
 *
 * Combined view:
 *   GET    /api/v1/timetable/timetable?academic_year_id= — get structures + slots
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
    order: number;
    created_at?: string;
    updated_at?: string;
}

export interface CreateTimeBlockPayload {
    day_of_week: number;
    period_name: string;
    start_time: string;
    end_time: string;
    is_break: boolean;
    academic_year_id?: string;
    order: number;
}

export interface UpdateTimeBlockPayload extends CreateTimeBlockPayload {
    academic_year_id: string;
}

export interface DeleteResult {
    deleted: boolean;
    deleted_count?: number;
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
    learning_area_id: string;
    teacher_id: string;
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
    session_status?: string | null;
    skip_reason?: string | null;
}

export interface SlotFilter {
    academic_year_id?: string;
    structure_id?: string;
    class_id?: string;
    teacher_id?: string;
    learning_area_id?: string;
}

export interface CreateSlotPayload {
    structure_id: string;
    class_id: string;
    learning_area_id: string;
    teacher_id: string;
    room_identifier?: string | null;
}

export interface BatchCreateSlotsPayload {
    slots: CreateSlotPayload[];
}

export interface UpdateSlotPayload {
    learning_area_id?: string;
    teacher_id?: string;
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
export async function listTimeBlocks(academicYearID: string): Promise<TimeBlock[]> {
    const qs = new URLSearchParams({ academic_year_id: academicYearID }).toString();
    return api.get<TimeBlock[]>(`/api/v1/timetable/structure?${qs}`);
}

/** Get a single time block by ID. */
export async function getTimeBlock(id: string): Promise<TimeBlock> {
    return api.get<TimeBlock>(`/api/v1/timetable/structure/${id}`);
}

/** Create a new time block. academic_year_id is optional (resolved server-side if omitted). */
export async function createTimeBlock(payload: CreateTimeBlockPayload): Promise<TimeBlock> {
    const qs = payload.academic_year_id ? `?academic_year_id=${payload.academic_year_id}` : "";
    return api.post<TimeBlock>(`/api/v1/timetable/structure${qs}`, payload);
}

/** Update an existing time block. */
export async function updateTimeBlock(
    id: string,
    payload: UpdateTimeBlockPayload
): Promise<TimeBlock> {
    return api.put<TimeBlock>(`/api/v1/timetable/structure/${id}`, payload);
}

/** Delete a time block by ID. */
export async function deleteTimeBlock(id: string): Promise<DeleteResult> {
    return api.delete<DeleteResult>(`/api/v1/timetable/structure/${id}`);
}

// ─── API Functions: Allocation Slots ──────────────────────────────────────

/** List slots with optional filters. academic_year_id is required. */
export async function listSlots(filters: SlotFilter): Promise<TimetableSlot[]> {
    const params = new URLSearchParams();
    if (filters.academic_year_id) params.set("academic_year_id", filters.academic_year_id);
    if (filters.structure_id) params.set("structure_id", filters.structure_id);
    if (filters.class_id) params.set("class_id", filters.class_id);
    if (filters.teacher_id) params.set("teacher_id", filters.teacher_id);
    if (filters.learning_area_id) params.set("learning_area_id", filters.learning_area_id);
    return api.get<TimetableSlot[]>(`/api/v1/timetable/slots?${params.toString()}`);
}

/** Create a single slot assignment. academic_year_id is optional (resolved server-side if omitted). */
export async function createSlot(payload: CreateSlotPayload): Promise<TimetableSlot> {
    return api.post<TimetableSlot>("/api/v1/timetable/slots", payload);
}

/** Batch-create slots (atomic). academic_year_id is required. */
export async function batchCreateSlots(
    payload: BatchCreateSlotsPayload,
    academicYearID: string
): Promise<TimetableSlot[]> {
    const qs = `?academic_year_id=${academicYearID}`;
    return api.post<TimetableSlot[]>(`/api/v1/timetable/slots/batch${qs}`, payload);
}

/** Update a slot assignment. */
export async function updateSlot(id: string, payload: UpdateSlotPayload): Promise<TimetableSlot> {
    return api.put<TimetableSlot>(`/api/v1/timetable/slots/${id}`, payload);
}

/** Delete a slot by ID. */
export async function deleteSlot(id: string): Promise<{ deleted: boolean }> {
    return api.delete<{ deleted: boolean }>(`/api/v1/timetable/slots/${id}`);
}

// ─── API Functions: Combined Timetable View ───────────────────────────────

/** Get combined timetable view (structures + slots) for an academic year. */
export async function getTimetable(academicYearID: string): Promise<{
    structures: TimeBlock[];
    slots: TimetableSlot[];
}> {
    const qs = new URLSearchParams({ academic_year_id: academicYearID }).toString();
    return api.get<{
        structures: TimeBlock[];
        slots: TimetableSlot[];
    }>(`/api/v1/timetable/timetable?${qs}`);
}
