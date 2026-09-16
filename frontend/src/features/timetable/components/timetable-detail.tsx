"use client";

import { TimetableGrid } from "./timetable-grid";
import { useTimeSlots } from "../hooks/use-time-slots";

type Props = {
    templateId: string;
};

export function TimetableDetail({ templateId }: Props) {
    const { data: slots = [], isLoading } = useTimeSlots(templateId);

    return (
        <div className="space-y-6 p-6">
            <h1 className="text-2xl font-semibold">Timetable Template</h1>
            <TimetableGrid slots={slots} isLoading={isLoading} templateId={templateId} />
        </div>
    );
}
