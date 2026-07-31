/**
 * Intercepted route — Attendance marking sheet.
 *
 * Slides out from the right when the user clicks "Mark Attendance"
 * on a timeline slot. On hard refresh, the full page at
 * /attendance/mark/[slot_id]/[date] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { AttendanceGrid } from "@/features/attendance/components/attendance-grid";
import { useSlotDetail } from "@/features/timetable-structure/hooks/use-timetable-structure";
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";

interface Props {
    params: Promise<{ slot_id: string; date: string }>;
}

export default function AttendanceMarkSheet({ params }: Props) {
    const router = useRouter();
    const { slot_id, date } = use(params);

    const { data: slot, isLoading, isError } = useSlotDetail(slot_id);

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-lg">
                <SheetHeader>
                    <SheetTitle>Mark Attendance</SheetTitle>
                    {isLoading ? (
                        <Skeleton className="h-4 w-64" />
                    ) : isError ? (
                        <SheetDescription>
                            Failed to load slot details. Please try again.
                        </SheetDescription>
                    ) : (
                        <SheetDescription>
                            {slot
                                ? `${slot.class_name} · ${slot.teacher_name ?? "No teacher assigned"} · ${slot.start_time?.slice(0, 5)}–${slot.end_time?.slice(0, 5)}`
                                : `${date}`}
                        </SheetDescription>
                    )}
                </SheetHeader>

                <div className="mt-6 flex-1 overflow-y-auto px-1 pb-6">
                    {isLoading ? (
                        <div className="space-y-3">
                            {Array.from({ length: 5 }).map((_, i) => (
                                <Skeleton key={i} className="h-10 w-full" />
                            ))}
                        </div>
                    ) : (
                        <AttendanceGrid
                            timetableSlotId={slot_id}
                            date={date}
                            onMarkedAttendance={() => router.back()}
                        />
                    )}
                </div>
            </SheetContent>
        </Sheet>
    );
}
