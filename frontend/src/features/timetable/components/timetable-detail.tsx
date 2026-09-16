"use client";

import { TimetableGrid } from "./timetable-grid";
import { useTimeSlots } from "../hooks/use-time-slots";
import type { TimeSlotResponse } from "../services/timetable-api";
import type { TimeSlotDraft } from "../types/timetable-template";

type Props = {
    templateId: string;
};

function toDraft(slot: TimeSlotResponse): TimeSlotDraft {
    return {
        id: slot.id,
        name: slot.name,
        start_time: slot.start_time,
        end_time: slot.end_time,
        is_instructional: slot.is_instructional,
    };
}

export function TimetableDetail({ templateId }: Props) {
    const { data: slots = [], isLoading } = useTimeSlots(templateId);

    return (
        <div className="space-y-6 p-6">
            <h1 className="text-2xl font-semibold">Timetable Template</h1>
            <TimetableGrid slots={slots.map(toDraft)} isLoading={isLoading} />
        </div>
    );
}
