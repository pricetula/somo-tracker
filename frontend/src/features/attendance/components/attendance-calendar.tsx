"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { startOfMonth, endOfMonth, startOfWeek, endOfWeek, format } from "date-fns";
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { useCalendarStatus } from "@/features/attendance/hooks/use-attendance";
import { useMe } from "@/hooks/use-auth";
import dynamic from "next/dynamic";
import { CalendarSkeleton } from "./calendar-skeleton";
import { StatusDot } from "./status-dot";

const Calendar = dynamic(() => import("@/components/shared/calendar").then((mod) => mod.Calendar), {
    ssr: false,
    loading: () => <CalendarSkeleton />,
});
export interface DaySummary {
    present: number;
    absent: number;
    late: number;
    excused: number;
    total: number;
}
export type DayStatus = "none" | "green" | "yellow" | "red";
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
function dateToKey(date: Date): string {
    return format(date, "yyyy-MM-dd");
}
interface AttendanceCalendarProps {
    /** School ID for fetching attendance status data. If omitted, derived from auth. */
    schoolId?: string;
    /** Optional CSS class for the outer wrapper. */
    className?: string;
    /** Legacy prop for backwards compatibility (previously passed by dashboard). */
    attendanceRateMap?: Record<string, number>;
}

export function AttendanceCalendar({ schoolId: propSchoolId }: AttendanceCalendarProps) {
    const statusTypes = [
        { label: "Complete", color: "bg-primary" },
        { label: "Partially recorded", color: "bg-blue-300" },
        { label: "Not recorded", color: "bg-rose-500" },
    ];
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
        <Card className="flex flex-col">
            <CardHeader className="flex items-center justify-between pb-0">
                <CardTitle>Attendance recorded</CardTitle>
            </CardHeader>
            <CardContent className="flex h-90 justify-center pb-0">
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
            </CardContent>
            <CardFooter className="flex justify-center gap-4">
                {statusTypes.map((s) => (
                    <span key={s.color}>
                        <span className={`mr-2 inline-block h-2 w-2 rounded-full ${s.color}`} />
                        <span>{s.label}</span>
                    </span>
                ))}
            </CardFooter>
        </Card>
    );
}
