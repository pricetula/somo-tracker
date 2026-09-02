/**
 * Full page — timetable/<trackId>/attendance?date=YYYY-MM-DD
 * Read-only attendance view from unified backend query.
 */
"use client";

import { use } from "react";
import { MarkedAllocationView } from "@/features/timetable";

interface Props {
    params: Promise<{ trackId: string }>;
    searchParams: Promise<{ date?: string }>;
}

export default function TimetableAllocationAttendancePage({ params, searchParams }: Props) {
    const { trackId } = use(params);
    const { date } = use(searchParams);

    if (!date) {
        return (
            <div className="text-muted-foreground p-6 text-sm">
                Missing date parameter. Add ?date=YYYY-MM-DD to the URL.
            </div>
        );
    }

    return (
        <div className="bg-background mx-auto max-w-2xl">
            <MarkedAllocationView allocationId={trackId} date={date} />
        </div>
    );
}
