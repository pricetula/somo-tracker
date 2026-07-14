/**
 * AttendanceRegisterContainer — reads query params and renders the roster
 * for a specific slot/date. Used as drill-down from admin dashboard.
 */

"use client";

import { useSearchParams } from "next/navigation";
import { TeacherAttendanceRoster } from "./teacher-attendance-roster";

interface AttendanceRegisterContainerProps {
    role: string;
}

export function AttendanceRegisterContainer({ role }: AttendanceRegisterContainerProps) {
    const searchParams = useSearchParams();
    const slotId = searchParams.get("slot_id");
    const date = searchParams.get("date") ?? undefined;

    if (!slotId) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No slot selected. Please navigate from the attendance dashboard.</p>
            </div>
        );
    }

    // Admins have elevated scope (no same-day restriction)
    const isAdmin = role === "SCHOOL_ADMIN" || role === "SYSTEM_ADMIN";

    return (
        <div className="space-y-6">
            <TeacherAttendanceRoster timetableSlotId={slotId} date={date} isLocked={!isAdmin} />
        </div>
    );
}
