"use client";

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
    ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useDayOfWeekSummaries } from "@/features/attendance/hooks/use-day-of-week-summaries";

const chartConfig = {
    absent_count: {
        label: "Absent",
    },
    late_count: {
        label: "Late",
    },
    excused_count: {
        label: "Excused",
    },
} satisfies ChartConfig;

interface WeekdayAttendanceExceptionsChartProps {
    /** Optional class UUID; when omitted the "All" school-wide rollup is shown. */
    classId?: string;
}

/**
 * Shadcn UI stacked BarChart for the School Administrator dashboard.
 *
 * Renders attendance exceptions (Absent, Late, Excused) stacked per weekday
 * (Monday → Friday) for the current academic year. Absent is rendered in the
 * destructive (red) semantic colour so the truancy/watch-list signal dominates
 * the stack; late and excused use chart palette tokens.
 *
 * Backed by GET /api/v1/attendance/day-of-week-summaries?class_id=….
 */
export function WeekdayAttendanceExceptionsChart({
    classId,
}: WeekdayAttendanceExceptionsChartProps) {
    const { data, isLoading, isError, error: _error } = useDayOfWeekSummaries(classId);

    if (isLoading) {
        return (
            <Card className="flex flex-col">
                <CardHeader className="flex items-center justify-between pb-0">
                    <CardTitle>Weekday attendance exceptions</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-1 items-center pb-0">
                    <Skeleton className="h-28 w-full" />
                </CardContent>
            </Card>
        );
    }

    if (isError) {
        return (
            <Card className="flex flex-col">
                <CardHeader className="flex items-center justify-between pb-0">
                    <CardTitle>Weekday attendance exceptions</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-1 items-center pb-0">
                    <div className="text-foreground text-center">
                        Failed to load weekday attendance exceptions.
                    </div>
                </CardContent>
            </Card>
        );
    }

    const items = data?.data ?? [];

    if (items.length === 0) return null;

    return (
        <Card className="flex flex-col">
            <CardHeader className="flex items-center justify-between pb-0">
                <CardTitle>Weekday attendance exceptions</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-1 items-center pb-0">
                <ChartContainer config={chartConfig} className="h-28 w-full">
                    <BarChart
                        data={items}
                        margin={{ top: 0, right: 0, left: 0, bottom: 0 }}
                        barSize={34}
                    >
                        <CartesianGrid strokeDasharray="3 3" vertical={false} />
                        <XAxis
                            dataKey="day_name"
                            tickLine={false}
                            axisLine={false}
                            tickMargin={8}
                        />
                        <YAxis hide tickLine={false} axisLine={false} tickMargin={2} />
                        <ChartTooltip content={<ChartTooltipContent />} />
                        <Bar
                            dataKey="absent_count"
                            fill="var(--destructive)"
                            stackId="exceptions"
                            radius={[0, 0, 0, 0]}
                        />
                        <Bar
                            dataKey="excused_count"
                            fill="var(--chart-2)"
                            stackId="exceptions"
                            radius={[0, 0, 0, 0]}
                        />
                        <Bar
                            dataKey="late_count"
                            fill="var(--chart-1)"
                            stackId="exceptions"
                            radius={[4, 4, 0, 0]}
                        />
                    </BarChart>
                </ChartContainer>
            </CardContent>
        </Card>
    );
}
