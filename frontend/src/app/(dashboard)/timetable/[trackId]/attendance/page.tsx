/**
 * Attendance marking full page — fallback for hard refresh of the
 * intercepted modal. Reuses the same AttendanceMarkingForm so behavior
 * stays consistent between the sheet and the standalone page.
 */
"use client";

import { use } from "react";
import { AttendanceMarkingForm } from "@/features/timetable";

interface Props {
    params: Promise<{ trackId: string }>;
    searchParams: Promise<{ date?: string }>;
}

export default function AttendanceMarkingPage({ params, searchParams }: Props) {
    const { trackId } = use(params);
    const { date } = use(searchParams);

    return (
        <div className="bg-background mx-auto max-w-2xl p-6">
            <h1 className="text-foreground mb-4 text-lg font-semibold">Mark attendance</h1>
            <AttendanceMarkingForm allocationId={trackId} date={date ?? ""} />
        </div>
    );
}
