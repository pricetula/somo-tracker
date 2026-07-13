/**
 * Timetable Structure feature — type definitions.
 *
 * Types are sourced from src/lib/api/timetable-structure.ts — never redefine here.
 */

export type {
    TimeBlock,
    TimeBlockListResult,
    CreateTimeBlockPayload,
    BatchCreateTimeBlockPayload,
    ReplicateDayPayload,
    DeleteResult,
    TimetableSlot,
    EnrichedSlot,
    SlotListResult,
    EnrichedSlotListResult,
    CreateSlotPayload,
    BatchCreateSlotsPayload,
    UpdateSlotPayload,
} from "@/lib/api/timetable-structure";

export {
    getDayName,
    getDayNameShort,
    DAY_NAMES,
    DAY_NAMES_SHORT,
} from "@/lib/api/timetable-structure";
