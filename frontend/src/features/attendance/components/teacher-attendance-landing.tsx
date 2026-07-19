/**
 * TeacherAttendanceLanding — shows the teacher's current/next period roster.
 * Pure shadcn: no borders, no cards, flat layout.
 *
 * If the teacher has an upcoming or in-progress period today, renders the
 * TeacherAttendanceRoster directly so they can mark attendance immediately.
 * Otherwise shows a helpful message and links to history.
 */

"use client";

import { useMemo } from "react";
import Link from "next/link";
import { ClipboardList, Clock } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";

import { useMe } from "@/hooks/use-auth";
import { useAcademicYears } from "@/features/academic-terms/hooks/use-academic-terms";
import { useEnrichedSlotList } from "@/features/timetable-structure/hooks/use-timetable-structure";

import { TeacherAttendanceRoster } from "./teacher-attendance-roster";
import { AttendanceEmptyState } from "./attendance-empty-state";
import { TeacherHistoryView } from "./teacher-history-view";

/** Returns 1=Monday … 7=Sunday. */
function getCurrentDayOfWeek(): number {
    const jsDay = new Date().getDay();
    return jsDay === 0 ? 7 : jsDay;
}

/** Returns today's date as YYYY-MM-DD. */
function todayISO(): string {
    return new Date().toISOString().split("T")[0];
}

/** Returns the current time as "HH:MM". */
function currentHHMM(): string {
    return new Date().toLocaleTimeString("en-GB", {
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
    });
}

export function TeacherAttendanceLanding() {
    const { data: me, isLoading: meLoading } = useMe();
    const { data: academicYears, isLoading: yearsLoading } = useAcademicYears();

    const currentYear = useMemo(
        () => academicYears?.items?.find((y) => y.is_current),
        [academicYears]
    );

    const enabled = !!currentYear?.id && !!me?.user_id;

    const { data: slotsData, isLoading: slotsLoading } = useEnrichedSlotList(
        currentYear?.id ?? "",
        enabled ? { mode: "teacher", id: me!.user_id } : undefined
    );

    const isLoading = meLoading || yearsLoading || slotsLoading;

    // Determine the current or next period for today
    const currentSlot = useMemo(() => {
        if (!slotsData?.items) return null;

        const today = getCurrentDayOfWeek();
        const now = currentHHMM();

        // Today's non-break slots sorted by start time
        const todaySlots = slotsData.items
            .filter((s) => !s.is_break && s.day_of_week === today)
            .sort((a, b) => a.start_time.localeCompare(b.start_time));

        if (todaySlots.length === 0) return null;

        // Find the current/next slot
        for (const slot of todaySlots) {
            if (now >= slot.start_time && now < slot.end_time) {
                return slot; // In-progress slot
            }
        }

        // No in-progress slot — find the next one
        for (const slot of todaySlots) {
            if (now < slot.start_time) {
                return slot; // Upcoming slot
            }
        }

        // All slots for today have passed — return the last one
        // (so the teacher can still see it, though it will be locked)
        return todaySlots[todaySlots.length - 1];
    }, [slotsData]);

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-64 w-full" />
            </div>
        );
    }

    if (!me) {
        return (
            <div className="text-destructive bg-destructive/10 p-4">
                Unable to verify your session. Please log in again.
            </div>
        );
    }

    if (!currentYear) {
        return (
            <AttendanceEmptyState
                icon={Clock}
                title="No active academic year"
                description="An academic year must be set as current before attendance can be marked."
            />
        );
    }

    // No current/upcoming slot found for today
    if (!currentSlot) {
        const todayName = [
            "Sunday",
            "Monday",
            "Tuesday",
            "Wednesday",
            "Thursday",
            "Friday",
            "Saturday",
        ][getCurrentDayOfWeek()];

        return (
            <div className="space-y-6">
                <h1 className="text-foreground text-2xl font-bold">Attendance</h1>
                <AttendanceEmptyState
                    icon={ClipboardList}
                    title="No periods scheduled for today"
                    description={
                        todayName === "Saturday" || todayName === "Sunday"
                            ? "It's the weekend! No classes are scheduled."
                            : `You don't have any classes scheduled for ${todayName}.`
                    }
                >
                    <div className="flex gap-3">
                        <Button variant="outline" size="sm" asChild>
                            <Link href="/attendance/history">View history</Link>
                        </Button>
                        <Button variant="outline" size="sm" asChild>
                            <Link href="/timetable">My timetable</Link>
                        </Button>
                    </div>
                </AttendanceEmptyState>

                {/* Show history below for quick access */}
                <TeacherHistoryView />
            </div>
        );
    }

    const slotDate = todayISO();
    const todayName = [
        "Sunday",
        "Monday",
        "Tuesday",
        "Wednesday",
        "Thursday",
        "Friday",
        "Saturday",
    ][getCurrentDayOfWeek()];

    const periodInfo =
        currentSlot.start_time <= currentHHMM() && currentSlot.end_time > currentHHMM()
            ? "In progress"
            : "Upcoming";

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-foreground text-2xl font-bold">Attendance</h1>
                    <p className="text-muted-foreground text-sm">
                        {todayName}, {slotDate}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <span className="bg-muted text-muted-foreground rounded-md px-2 py-1 text-xs font-medium">
                        {periodInfo}
                    </span>
                    <Button variant="outline" size="sm" asChild>
                        <Link href="/attendance/history">Full history</Link>
                    </Button>
                </div>
            </div>

            <TeacherAttendanceRoster timetableSlotId={currentSlot.id} date={slotDate} />
        </div>
    );
}
