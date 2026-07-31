/**
 * Intercepted route — Attendance marking sheet.
 *
 * Slides out from the right when the user clicks "Mark Attendance"
 * on a timeline slot. On hard refresh, the full page at
 * /attendance/mark/[slot_id]/[date] takes over.
 */

"use client";

import { useRouter } from "next/navigation";
import { SessionList } from "@/features/attendance/components/session-list";
import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
    SheetDescription,
} from "@/components/ui/sheet";

export default function AttendanceSessions() {
    const router = useRouter();
    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-lg">
                <SheetHeader>
                    <SheetTitle>Attendance Sessions</SheetTitle>
                    <SheetDescription>List attendance sessions</SheetDescription>
                </SheetHeader>

                <SessionList />
            </SheetContent>
        </Sheet>
    );
}
