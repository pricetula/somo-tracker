/**
 * AttendanceRegisterSheet — side sheet wrapper for the attendance register.
 *
 * Renders the full register (teacher-style roster) inside a sliding sheet
 * so admins can mark attendance without leaving the dashboard table.
 */

"use client";

import { useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AttendanceRegisterContainer } from "@/features/attendance/components/attendance-register-container";

interface AttendanceRegisterSheetProps {
    role: string;
    slotId: string;
    date?: string;
}

export function AttendanceRegisterSheet({ role, slotId, date }: AttendanceRegisterSheetProps) {
    const router = useRouter();

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent
                side="right"
                className="w-full overflow-y-auto data-[side=right]:sm:max-w-2xl"
            >
                <SheetHeader>
                    <SheetTitle>Attendance Register</SheetTitle>
                </SheetHeader>
                <div className="px-6 pb-6">
                    <AttendanceRegisterContainer role={role} slotId={slotId} date={date} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
