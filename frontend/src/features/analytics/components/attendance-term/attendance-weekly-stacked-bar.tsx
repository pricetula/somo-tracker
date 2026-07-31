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
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    present: {
        label: "Present",
        color: "#22c55e",
    },
    late: {
        label: "Late",
        color: "#3b82f6",
    },
    absent: {
        label: "Absent",
        color: "#ef4444",
    },
    excused: {
        label: "Excused",
        color: "#8b5cf6",
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
            <p className="text-foreground text-sm font-medium">
                Weekly Attendance Breakdown
                <GraphHelp>
                    Stacked bar chart breaking down weekly attendance into Present, Late, Absent,
                    and Excused categories.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} barCategoryGap="20%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="weekLabel" tickLine={false} tickMargin={8} axisLine={false} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar dataKey="present" stackId="a" fill="#22c55e" radius={[0, 0, 0, 0]} />
                    <Bar dataKey="late" stackId="a" fill="#3b82f6" radius={[0, 0, 0, 0]} />
                    <Bar dataKey="absent" stackId="a" fill="#ef4444" radius={[0, 0, 0, 0]} />
                    <Bar dataKey="excused" stackId="a" fill="#8b5cf6" radius={[0, 0, 0, 0]} />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
