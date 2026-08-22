"use client";

import { useMemo } from "react";
import { useTimeBlocks } from "./use-timetable-structure";
import { useEnrichedSlots } from "./use-timetable-slots";
import type {
    TimetableGridView,
    TimetableCellView,
    TimetableDayColumn,
    TimeBlock,
    EnrichedSlot,
} from "../types";
import { DAYS_OF_WEEK } from "../types";

interface UseTimetableGridOptions {
    academicYearId: string;
    filters?: {
        classId?: string;
        teacherId?: string;
        roomIdentifier?: string;
        date?: string;
    };
}

/**
 * Builds the complete timetable grid view model from structure blocks and allocated slots.
 * Handles all empty states and conflict detection.
 */
export function useTimetableGrid({ academicYearId, filters }: UseTimetableGridOptions) {
    const {
        data: structureData,
        isLoading: structureLoading,
        error: structureError,
    } = useTimeBlocks(academicYearId);
    const {
        data: slotsData,
        isLoading: slotsLoading,
        error: slotsError,
    } = useEnrichedSlots(academicYearId, filters);

    const gridView = useMemo((): TimetableGridView => {
        const timeBlocks = structureData?.items ?? [];
        const slots = slotsData?.items ?? [];

        // Determine empty state
        let emptyState: TimetableGridView["emptyState"] = "none";
        if (timeBlocks.length === 0) {
            emptyState = "no_structure";
        } else if (slots.length === 0 && (!filters || Object.values(filters).every((v) => !v))) {
            emptyState = "no_slots";
        } else if (slots.length === 0 && filters && Object.values(filters).some((v) => v)) {
            emptyState = "filtered_empty";
        }

        // Sort periods by start_time
        const periods = [...timeBlocks].sort((a, b) => a.start_time.localeCompare(b.start_time));

        // Group time blocks by day
        const blocksByDay = new Map<number, TimeBlock[]>();
        for (const block of periods) {
            const dayBlocks = blocksByDay.get(block.day_of_week) ?? [];
            dayBlocks.push(block);
            blocksByDay.set(block.day_of_week, dayBlocks);
        }

        // Create a lookup for slots by structure_id
        const slotsByStructureId = new Map<string, EnrichedSlot[]>();
        for (const slot of slots) {
            const existing = slotsByStructureId.get(slot.structure_id) ?? [];
            existing.push(slot);
            slotsByStructureId.set(slot.structure_id, existing);
        }

        // Build day columns
        const days: TimetableDayColumn[] = DAYS_OF_WEEK.map(({ value: day }) => {
            const dayBlocks = blocksByDay.get(day) ?? [];
            const cells: TimetableCellView[] = dayBlocks.map((structure) => {
                const structureSlots = slotsByStructureId.get(structure.id) ?? [];
                const slot = structureSlots[0]; // Take first slot if multiple (shouldn't happen with unique constraints)

                // Detect conflicts from session_status or backend error codes
                // session_status: "SUBMITTED" | "SKIPPED" | null
                // We'll also check if there are multiple slots for the same structure (conflict)
                const hasConflict = structureSlots.length > 1;
                const conflictMessage = hasConflict
                    ? "Multiple lessons assigned to this period"
                    : slot?.session_status === "SKIPPED"
                      ? `Skipped: ${slot.skip_reason ?? "No reason given"}`
                      : undefined;

                return {
                    structure,
                    slot,
                    hasConflict,
                    conflictMessage,
                };
            });

            return { day, cells };
        });

        return {
            periods,
            days,
            emptyState,
            filters: filters ?? {},
        };
    }, [structureData, slotsData, filters]);

    return {
        gridView,
        isLoading: structureLoading || slotsLoading,
        error: structureError ?? slotsError,
        timeBlocks: structureData?.items ?? [],
        slots: slotsData?.items ?? [],
    };
}
