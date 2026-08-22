import type { TimeBlock, EnrichedSlot } from "@/lib/api/timetable-structure";
// Re-export API types
export type { TimeBlock, EnrichedSlot } from "@/lib/api/timetable-structure";
// Local view types (not re-exported to avoid conflicts)
// export type { TimetableCell, TimetableDayColumn, TimetableGridView } from "./timetable-types";

/** Day of week constants (1=Monday, 7=Sunday) */
export const DAYS_OF_WEEK = [
    { value: 1, label: "Monday", short: "Mon" },
    { value: 2, label: "Tuesday", short: "Tue" },
    { value: 3, label: "Wednesday", short: "Wed" },
    { value: 4, label: "Thursday", short: "Thu" },
    { value: 5, label: "Friday", short: "Fri" },
    { value: 6, label: "Saturday", short: "Sat" },
] as const;

export type DayOfWeek = (typeof DAYS_OF_WEEK)[number]["value"];

/** View model for a single grid cell */
export interface TimetableCellView {
    // Renamed to avoid conflict with component export

    /** The structural time block (period definition) */
    structure: TimeBlock;
    /** The allocated slot if one exists for the current filter */
    slot?: EnrichedSlot;
    /** Whether this cell has a conflict (teacher/room/class double-booked) */
    hasConflict?: boolean;
    /** Conflict message if any */
    conflictMessage?: string;
}

/** View model for a full day column */
export interface TimetableDayColumn {
    // Renamed to avoid conflict with component export

    day: DayOfWeek;
    cells: TimetableCellView[];
}

/** View model for the entire week grid */
export interface TimetableGridView {
    /** Periods (time blocks) as rows — sorted by start_time */
    periods: TimeBlock[];
    /** Days as columns */
    days: TimetableDayColumn[];
    /** Whether we're in an empty state */
    emptyState: "no_structure" | "no_slots" | "filtered_empty" | "none";
    /** Active filters */
    filters: TimetableFilters;
}

/** Client-side filters for the timetable view */
export interface TimetableFilters {
    classId?: string;
    teacherId?: string;
    roomIdentifier?: string;
    date?: string; // ISO date string for session_status enrichment
}

/** Payload for creating a slot assignment via modal */
export interface CreateSlotPayload {
    structureId: string;
    classId: string;
    learningAreaId: string;
    teacherId: string;
    roomIdentifier?: string;
}

/** Internal form state for slot assignment (allows empty strings during editing) */
export interface SlotFormState {
    structureId: string;
    classId: string;
    learningAreaId: string;
    teacherId: string;
    roomIdentifier: string;
}

/** Payload for updating a slot assignment */
export interface UpdateSlotPayload {
    learningAreaId?: string;
    teacherId?: string;
    roomIdentifier?: string;
}

/** Structure builder step types */
export type StructureBuilderStep = "periods" | "review" | "complete";

/** Period form data for structure builder */
export interface PeriodFormData {
    dayOfWeek: DayOfWeek;
    periodName: string;
    startTime: string;
    endTime: string;
    isBreak: boolean;
}

/** Replicate day payload for structure builder */
export interface ReplicateDayFormData {
    sourceDay: DayOfWeek;
    targetDays: DayOfWeek[];
}
