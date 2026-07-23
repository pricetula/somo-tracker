/**
 * AttendanceMarkPage — shared component used by both the full-page route
 * and the intercepted modal sheet.
 *
 * Fetches the enriched slot details via React Query and renders the AttendanceGrid.
 */
"use client";

import { AttendanceGrid } from "./attendance-grid";
import { Skeleton } from "@/components/ui/skeleton";
import { useSlotDetail } from "@/features/timetable-structure/hooks/use-timetable-structure";

interface Props {
    slotId: string;
    date: string;
}

export function AttendanceMarkPage({ slotId, date }: Props) {
    const { data: slot, isLoading, isError } = useSlotDetail(slotId);

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-4 w-64" />
                <div className="space-y-3">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <Skeleton key={i} className="h-10 w-full" />
                    ))}
                </div>
            </div>
        );
    }

    if (isError) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md p-4 text-sm">
                Failed to load slot details. Please try again.
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-xl font-bold">{slot?.period_name ?? "Mark Attendance"}</h1>
                {slot && (
                    <p className="text-muted-foreground text-sm">
                        {slot.class_name}
                        {slot.teacher_name ? ` · ${slot.teacher_name}` : ""}
                        {` · ${slot.start_time?.slice(0, 5)}–${slot.end_time?.slice(0, 5)}`}
                    </p>
                )}
            </div>
            <AttendanceGrid timetableSlotId={slotId} date={date} />
        </div>
    );
}
