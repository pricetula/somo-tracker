"use client";

import { useRouter } from "next/navigation";
import { format } from "date-fns";
import { useCalendarStatus } from "@/features/attendance/hooks/use-attendance";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { Skeleton } from "@/components/ui/skeleton";

const DEFAULT_DATE_RANGE = {
    start: format(new Date(new Date().setMonth(new Date().getMonth() - 1)), "yyyy-MM-dd"),
    end: format(new Date(), "yyyy-MM-dd"),
};

interface CalendarStatusProps {
    schoolId?: string;
    startDate?: string;
    endDate?: string;
}

export function CalendarStatusView({ schoolId, startDate, endDate }: CalendarStatusProps) {
    const router = useRouter();
    const {
        data: statusData,
        isLoading,
        isError,
    } = useCalendarStatus(
        startDate || DEFAULT_DATE_RANGE.start,
        endDate || DEFAULT_DATE_RANGE.end,
        schoolId
    );

    if (isLoading) {
        return (
            <div className="space-y-4">
                <h1 className="text-2xl font-bold">Calendar Status</h1>
                <Skeleton className="h-40 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="space-y-4">
                <h1 className="text-destructive text-2xl font-bold">
                    Failed to load calendar status
                </h1>
            </div>
        );
    }

    const dayStatusData = statusData?.items || [];
    const displayStart = startDate || DEFAULT_DATE_RANGE.start;
    const displayEnd = endDate || DEFAULT_DATE_RANGE.end;

    return (
        <div className="mx-auto max-w-7xl space-y-6">
            <div className="flex items-center justify-between p-4 text-sm">
                <p className="text-muted-foreground">
                    Calendar Status: {displayStart} to {displayEnd}
                </p>
                <button
                    className="bg-primary text-primary-foreground hover:bg-primary/90 px-4 py-2 text-sm"
                    onClick={() => {
                        const newStart = new Date(displayStart);
                        newStart.setMonth(newStart.getMonth() - 1);
                        router.replace(
                            `/attendance/calendar-status?start_date=${format(newStart, "yyyy-MM-dd")}&end_date=${displayEnd}`
                        );
                    }}
                >
                    Back a Month
                </button>
            </div>

            <div className="bg-background rounded-lg border p-4">
                <ResponsiveContainer width="100%" height={300}>
                    <BarChart
                        data={dayStatusData}
                        margin={{ top: 20, right: 30, left: 0, bottom: 5 }}
                    >
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="date" />
                        <YAxis />
                        <Tooltip
                            formatter={(value, name) => {
                                if (name === "status") {
                                    return [value, "Status"];
                                }
                                return [value, name];
                            }}
                        />
                        {dayStatusData.map((day, index) => (
                            <Bar
                                key={index}
                                dataKey="status"
                                fill={
                                    day.status === "green"
                                        ? "hsl(174 77% 67%)"
                                        : day.status === "yellow"
                                          ? "hsl(42 96% 52%)"
                                          : day.status === "red"
                                            ? "hsl(0 84% 60%)"
                                            : "hsl(220 10% 55%)"
                                }
                            />
                        ))}
                    </BarChart>
                </ResponsiveContainer>
            </div>
        </div>
    );
}
