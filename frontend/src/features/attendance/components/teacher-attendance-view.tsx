/**
 * TeacherAttendanceView — finds the teacher's current/next period and renders
 * the attendance roster. Also fetches teacher's timetable slots for today.
 */

"use client";

import { useMe } from "@/hooks/use-auth";
import { TeacherAttendanceRoster } from "./teacher-attendance-roster";

// TODO: Replace with actual API call to get the teacher's current/next slot
// This is a stub that demonstrates the pattern. The real implementation
// should call GET /api/v1/timetable-slots?teacher_id=:id&date=today&is_break=false
// and find the one matching the current time, then pass it to TeacherAttendanceRoster.

const MOCK_SLOT_ID = "00000000-0000-0000-0000-000000000000";

export function TeacherAttendanceView() {
    const { data: me, isLoading } = useMe();

    if (isLoading) {
        return <div className="text-muted-foreground py-8 text-center">Loading...</div>;
    }

    if (!me) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Unable to verify your session. Please log in again.
            </div>
        );
    }

    // TODO: fetch today's timetable slots for this teacher
    // const { data: slots } = useTeacherTodaySlots(me.user_id);
    // if (!slots || slots.length === 0) → "No class in session right now."
    // const currentSlot = findCurrentSlot(slots);
    // return <TeacherAttendanceRoster timetableSlotId={currentSlot.id} />;

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance</h1>
            <TeacherAttendanceRoster timetableSlotId={MOCK_SLOT_ID} />
        </div>
    );
}
