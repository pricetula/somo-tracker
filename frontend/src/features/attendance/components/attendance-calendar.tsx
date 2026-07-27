"use client";

// import { Calendar } from "@/components/shared/calendar";
import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";

// Disable SSR for the Calendar component
const Calendar = dynamic(() => import("@/components/shared/calendar").then((mod) => mod.Calendar), {
    ssr: false,
    loading: () => (
        <div className="w-92 rounded-md p-3">
            {/* Header: Month name and navigation arrows */}
            <div className="flex items-center justify-between px-1 pt-1 pb-4">
                <Skeleton className="h-7 w-7 rounded-md" />
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-7 w-7 rounded-md" />
            </div>

            {/* Weekday Labels (Su Mo Tu We Th Fr Sa) */}
            <div className="mb-2 grid grid-cols-7 gap-1">
                {Array.from({ length: 7 }).map((_, i) => (
                    <div key={i} className="flex h-8 items-center justify-center">
                        <Skeleton className="h-3 w-4" />
                    </div>
                ))}
            </div>

            {/* Days Grid (5 weeks x 7 days) */}
            <div className="grid grid-cols-7 gap-2">
                {Array.from({ length: 35 }).map((_, i) => (
                    <div key={i} className="flex h-8 items-center justify-center">
                        <Skeleton className="h-8 w-8 rounded-md" />
                    </div>
                ))}
            </div>
        </div>
    ), // Optional fallback
});

// ─── Types ────────────────────────────────────────────────────────────────

export interface DaySummary {
    present: number;
    absent: number;
    late: number;
    excused: number;
    total: number;
}
// ─── Component ────────────────────────────────────────────────────────────

export function AttendanceCalendar() {
    return (
        <section className="w-fit">
            <header>Attendace Calendar</header>
            <Calendar onDayClick={console.log} />
        </section>
    );
}
