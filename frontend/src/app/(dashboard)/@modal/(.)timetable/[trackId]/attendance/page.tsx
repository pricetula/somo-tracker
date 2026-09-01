/**
 * Intercepted route — Attendance marking sheet for a timetable slot.
 *
 * Opens from the timeline when clicking a lesson entry, displaying
 * student roll-call with status checkboxes and optional notes.
 */
"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { AttendanceMarkingForm } from "@/features/timetable";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

interface Props {
    params: Promise<{ trackId: string }>;
    searchParams: Promise<{ date?: string }>;
}

export default function AttendanceMarkingSheet({ params, searchParams }: Props) {
    const router = useRouter();
    const { trackId } = use(params);
    const { date } = use(searchParams);

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-xl">
                <SheetHeader>
                    <SheetTitle>Mark attendance</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pt-2 pb-6">
                    <AttendanceMarkingForm allocationId={trackId} date={date ?? ""} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
