/**
 * AttendanceWeeklyStackedBar — Stacked bar chart showing Present / Late / Absent / Excused per week.
 *
 * Visualisation: Weekly breakdown of attendance statuses.
 * Props: Array of daily attendance records grouped by week.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartLegend,
    ChartLegendContent,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    present: {
        label: "Present",
        color: "hsl(var(--chart-2))",
    },
    late: {
        label: "Late",
        color: "hsl(var(--chart-3))",
    },
    absent: {
        label: "Absent",
        color: "hsl(var(--chart-1))",
    },
    excused: {
        label: "Excused",
        color: "hsl(var(--chart-5))",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface WeeklyAttendanceRow {
    week: string;
    weekLabel: string;
    present: number;
    late: number;
    absent: number;
    excused: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceWeeklyStackedBarProps {
    data: WeeklyAttendanceRow[];
}

export function AttendanceWeeklyStackedBar({ data }: AttendanceWeeklyStackedBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No weekly attendance data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Weekly Attendance Breakdown</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} barCategoryGap="20%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="weekLabel" tickLine={false} tickMargin={8} axisLine={false} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar
                        dataKey="present"
                        stackId="a"
                        fill="var(--color-present)"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar
                        dataKey="late"
                        stackId="a"
                        fill="var(--color-late)"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar
                        dataKey="absent"
                        stackId="a"
                        fill="var(--color-absent)"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar
                        dataKey="excused"
                        stackId="a"
                        fill="var(--color-excused)"
                        radius={[0, 0, 0, 0]}
                    />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function AttendanceWeeklyStackedBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
