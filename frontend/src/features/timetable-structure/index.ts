/**
 * Timetable Structure feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { StructurePage } from "./components/structure-page";
export { BlueprintDialog } from "./components/blueprint-dialog";
export { TemplateMenu } from "./components/template-menu";
export { TimetableSlotGrid } from "./components/timetable-slot-grid";
export { AddSlotDialog } from "./components/add-slot-dialog";

export {
    useTimeBlockList,
    useTimeBlockListByDay,
    useCreateTimeBlock,
    useBatchCreateTimeBlocks,
    useReplicateDay,
    useUpdateTimeBlock,
    useDeleteTimeBlock,
    useSlotList,
    useEnrichedSlotList,
    useSlotDetail,
    useCreateSlot,
    useBatchCreateSlots,
    useUpdateSlot,
    useDeleteSlot,
    timetableStructureKeys,
    timetableSlotKeys,
} from "./hooks/use-timetable-structure";

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
} from "./types";
export { getDayName, getDayNameShort, DAY_NAMES, DAY_NAMES_SHORT } from "./types";
