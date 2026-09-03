/**
 * Intercepted route — right-side sheet for timetable/<trackId>/attendance?date=...
 * Reuses the existing [trackId] dynamic param naming.
 */
"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { MarkedAllocationView } from "@/features/timetable";
import { Sheet, SheetContent } from "@/components/ui/sheet";

interface Props {
    params: Promise<{ trackId: string }>;
    searchParams: Promise<{ date?: string }>;
}

export default function TimetableAllocationAttendanceSheet({ params, searchParams }: Props) {
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
                {date ? (
                    <MarkedAllocationView allocationId={trackId} date={date} />
                ) : (
                    <p className="text-muted-foreground p-6 text-sm">Missing date parameter.</p>
                )}
            </SheetContent>
        </Sheet>
    );
}
