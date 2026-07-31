"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { startOfMonth, endOfMonth, startOfWeek, endOfWeek, format } from "date-fns";
import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useCalendarStatus } from "@/features/attendance/hooks/use-attendance";
import { useMe } from "@/hooks/use-auth";

// Disable SSR for the Calendar component (it uses browser APIs)
const Calendar = dynamic(() => import("@/components/shared/calendar").then((mod) => mod.Calendar), {
    ssr: false,
    loading: () => <CalendarSkeleton />,
});

// ─── Types ────────────────────────────────────────────────────────────────

export interface DaySummary {
    present: number;
    absent: number;
    late: number;
    excused: number;
    total: number;
}

export type DayStatus = "none" | "green" | "yellow" | "red";

// ─── Helpers ──────────────────────────────────────────────────────────────

/** Compute the visible date range for a given month, including leading/trailing
 *  days from adjacent months that appear in the calendar grid. */
function getVisibleRange(year: number, month: number): { start: string; end: string } {
    const monthDate = new Date(year, month, 1);
    const monthStart = startOfMonth(monthDate);
    const monthEnd = endOfMonth(monthDate);
    // Include full weeks (Sunday-based) so leading/trailing days are covered
    const gridStart = startOfWeek(monthStart, { weekStartsOn: 0 });
    const gridEnd = endOfWeek(monthEnd, { weekStartsOn: 0 });
    return {
        start: format(gridStart, "yyyy-MM-dd"),
        end: format(gridEnd, "yyyy-MM-dd"),
    };
}

/** Convert a date to a string key for the status map. */
function dateToKey(date: Date): string {
    return format(date, "yyyy-MM-dd");
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

function CalendarSkeleton() {
    return (
        <div className="w-92 rounded-md p-3">
            <div className="flex items-center justify-between px-1 pt-1 pb-4">
                <Skeleton className="h-7 w-7 rounded-md" />
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-7 w-7 rounded-md" />
            </div>
            <div className="mb-2 grid grid-cols-7 gap-1">
                {Array.from({ length: 7 }).map((_, i) => (
                    <div key={i} className="flex h-8 items-center justify-center">
                        <Skeleton className="h-3 w-4" />
                    </div>
                ))}
            </div>
            <div className="grid grid-cols-7 gap-2">
                {Array.from({ length: 35 }).map((_, i) => (
                    <div key={i} className="flex h-8 items-center justify-center">
                        <Skeleton className="h-8 w-8 rounded-md" />
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Status Dot ───────────────────────────────────────────────────────────

interface StatusDotProps {
    status: DayStatus;
}

function StatusDot({ status }: StatusDotProps) {
    if (status === "none") return null;

    const colorClass = {
        green: "bg-green-500",
        yellow: "bg-yellow-500",
        red: "bg-red-500",
    }[status];

    return (
        <span
            className={cn(
                "absolute -bottom-0.5 left-1/2 h-1.5 w-1.5 -translate-x-1/2 rounded-full",
                colorClass
            )}
            aria-hidden="true"
        />
    );
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceCalendarProps {
    /** School ID for fetching attendance status data. If omitted, derived from auth. */
    schoolId?: string;
    /** Optional CSS class for the outer wrapper. */
    className?: string;
    /** Legacy prop for backwards compatibility (previously passed by dashboard). */
    attendanceRateMap?: Record<string, number>;
}

export function AttendanceCalendar({ schoolId: propSchoolId, className }: AttendanceCalendarProps) {
    const router = useRouter();
    const { data: me } = useMe();
    const schoolId = propSchoolId ?? me?.school_id ?? "";

    const today = new Date();
    const [currentMonth, setCurrentMonth] = React.useState(today);

    const year = currentMonth.getFullYear();
    const month = currentMonth.getMonth();
    const { start: startDate, end: endDate } = getVisibleRange(year, month);

    // Fetch calendar status using React Query (consistent with codebase pattern)
    const {
        data,
        isFetching,
        isError,
        error: queryError,
    } = useCalendarStatus(startDate, endDate, schoolId);

    // Build a map of date → status for O(1) lookup
    const statusMap = React.useMemo(() => {
        const map: Record<string, DayStatus> = {};
        if (data?.items) {
            for (const item of data.items) {
                map[item.date] = item.status as DayStatus;
            }
        }
        return map;
    }, [data]);

    // Day content render prop — shows a colored dot for status
    const dayContent = React.useCallback(
        (date: Date): React.ReactNode => {
            const key = dateToKey(date);
            const status = statusMap[key];
            if (!status) return null;
            return <StatusDot status={status} />;
        },
        [statusMap]
    );

    return (
        <section className={cn("w-fit", className)}>
            <header>Attendance Calendar</header>
            <Calendar
                month={currentMonth}
                onMonthChange={setCurrentMonth}
                onDayClick={(date) => {
                    const dateStr = format(date, "yyyy-MM-dd");
                    if (dateStr) {
                        router.push(`/attendance/${dateStr}`);
                    }
                }}
                disabled={[{ after: today }]}
                dayContent={dayContent}
                showOutsideDays={true}
            />
            {isFetching && (
                <p className="text-muted-foreground mt-1 text-xs" role="status">
                    Loading attendance status…
                </p>
            )}
            {isError && (
                <p className="text-destructive mt-1 text-xs" role="alert">
                    {queryError instanceof Error
                        ? queryError.message
                        : "Failed to load attendance status"}
                </p>
            )}
        </section>
    );
}
