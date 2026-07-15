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
    // Admins have no same-day restriction; teachers can edit today's records.
    const isAdmin = role === "SCHOOL_ADMIN" || role === "SYSTEM_ADMIN";
    const today = new Date().toISOString().split("T")[0];
    const isPastDate = Boolean(date && date !== today);

    return (
        <div className="space-y-6">
            <TeacherAttendanceRoster
                timetableSlotId={slotId}
                date={date}
                isLocked={!isAdmin && isPastDate}
            />
        </div>
    );
}
