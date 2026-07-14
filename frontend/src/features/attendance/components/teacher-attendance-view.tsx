/**
 * TeacherAttendanceView — finds the teacher's current/next period and renders
 * the attendance roster.
 *
 * Fetches the teacher's enriched timetable slots for today, filters to non-break
 * periods, and finds the one matching the current time (or the next upcoming
 * period). Falls back to a "No class in session" state if none found.
 */

"use client";

import { useMemo } from "react";
import { useMe } from "@/hooks/use-auth";
import { useEnrichedSlotList } from "@/features/timetable-structure/hooks/use-timetable-structure";
import { useAcademicYears } from "@/features/academic-terms/hooks/use-academic-terms";
import Link from "next/link";
import { AlertCircle, School } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TeacherAttendanceRoster } from "./teacher-attendance-roster";
import { AttendanceEmptyState } from "./attendance-empty-state";
import { Skeleton } from "@/components/ui/skeleton";

function getCurrentDayOfWeek(): number {
    // JS getDay(): 0=Sun, 1=Mon ... 6=Sat
    // Backend: 1=Mon ... 7=Sun
    const jsDay = new Date().getDay();
    return jsDay === 0 ? 7 : jsDay;
}

function getCurrentTimeString(): string {
    return new Date().toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" });
}

function formatDate(date: Date): string {
    return date.toISOString().split("T")[0];
}

export function TeacherAttendanceView() {
    const { data: me, isLoading: meLoading } = useMe();
    const today = formatDate(new Date());
    const currentDay = getCurrentDayOfWeek();

    // Fetch current academic year (which gives us terms with is_current)
    const { data: academicYears, isLoading: yearsLoading } = useAcademicYears();
    const currentYear = useMemo(
        () => academicYears?.items?.find((y) => y.is_current),
        [academicYears]
    );

    // Fetch teacher's enriched slots for the current academic year
    const { data: slotsData, isLoading: slotsLoading } = useEnrichedSlotList(
        currentYear?.id ?? "",
        me ? { mode: "teacher", id: me.user_id } : undefined
    );

    const isLoading = meLoading || yearsLoading || slotsLoading;

    // Find today's non-break periods and determine current/next slot
    const currentSlot = useMemo(() => {
        if (!slotsData?.items?.length) return null;

        const todaySlots = slotsData.items.filter(
            (s) => s.day_of_week === currentDay && !s.is_break
        );

        if (todaySlots.length === 0) return null;

        const now = getCurrentTimeString();

        // Find the current slot (start_time <= now < end_time), or the next upcoming
        const current = todaySlots.find((s) => s.start_time <= now && s.end_time > now);
        if (current) return current;

        // No active slot — find the next upcoming one
        const upcoming = todaySlots
            .filter((s) => s.start_time > now)
            .sort((a, b) => a.start_time.localeCompare(b.start_time));
        if (upcoming.length > 0) return upcoming[0];

        // All slots have passed — return last one of the day (past)
        return todaySlots[todaySlots.length - 1];
    }, [slotsData, currentDay]);

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <div className="space-y-3">
                    <Skeleton className="h-8 w-full" />
                    {Array.from({ length: 6 }).map((_, i) => (
                        <Skeleton key={i} className="h-10 w-full" />
                    ))}
                </div>
            </div>
        );
    }

    if (!me) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Unable to verify your session. Please log in again.
            </div>
        );
    }

    if (!currentYear) {
        return (
            <AttendanceEmptyState
                icon={AlertCircle}
                title="No active academic year"
                description="An academic year must be set as current before attendance can be marked."
            >
                <Button variant="outline" size="sm" asChild>
                    <Link href="/settings">Contact school admin</Link>
                </Button>
            </AttendanceEmptyState>
        );
    }

    if (!currentSlot) {
        return (
            <AttendanceEmptyState
                icon={School}
                title="No class in session right now"
                description="Your timetable for today doesn't have any scheduled periods, or all periods have ended for the day."
            >
                <Button variant="outline" size="sm" asChild>
                    <Link href="/timetable">View my timetable</Link>
                </Button>
            </AttendanceEmptyState>
        );
    }

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance</h1>
            <div className="text-muted-foreground text-sm">
                {currentSlot.period_name} &middot; {currentSlot.class_name}
                {currentSlot.start_time > getCurrentTimeString() && (
                    <span className="ml-2 italic">
                        (next period starting at {currentSlot.start_time})
                    </span>
                )}
            </div>
            <TeacherAttendanceRoster timetableSlotId={currentSlot.id} date={today} />
        </div>
    );
}
