/**
 * DayOfWeekBar — Bar chart of average attendance by day of week.
 *
 * Visualisation: Identifies "Friday slump" patterns.
 * Props: Array of { dayName, averageRate } for Mon-Fri.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    rate: {
        label: "Avg Attendance",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface DayOfWeekAverage {
    dayName: string;
    averageRate: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface DayOfWeekBarProps {
    data: DayOfWeekAverage[];
}

export function DayOfWeekBar({ data }: DayOfWeekBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No day-of-week data available.
            </p>
        );
    }

    const order = ["Mon", "Tue", "Wed", "Thu", "Fri"];
    const sorted = [...data].sort((a, b) => order.indexOf(a.dayName) - order.indexOf(b.dayName));

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Average Attendance by Day of Week
                <GraphHelp>
                    Bar chart showing average attendance rate for each day of the week. Helps
                    identify patterns such as a &ldquo;Friday slump&rdquo;.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={sorted}>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="dayName" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="averageRate" fill="#22c55e" radius={[4, 4, 0, 0]} barSize={40} />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
