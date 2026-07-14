/**
 * AttendanceRegisterContainer — renders the roster for a specific slot/date.
 * `slotId` is required (from path param), `date` is optional (from query).
 */

import { TeacherAttendanceRoster } from "./teacher-attendance-roster";

interface AttendanceRegisterContainerProps {
    role: string;
    slotId: string;
    date?: string;
}

export function AttendanceRegisterContainer({
    role,
    slotId,
    date,
}: AttendanceRegisterContainerProps) {
    // Admins have elevated scope (no same-day restriction)
    const isAdmin = role === "SCHOOL_ADMIN" || role === "SYSTEM_ADMIN";

    return (
        <div className="space-y-6">
            <TeacherAttendanceRoster timetableSlotId={slotId} date={date} isLocked={!isAdmin} />
        </div>
    );
}
