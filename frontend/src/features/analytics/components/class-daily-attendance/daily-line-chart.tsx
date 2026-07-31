/**
 * DailyLineChart — Time-series line chart of daily attendance rate.
 *
 * Visualisation: Daily attendance rate with optional school holiday markers.
 * Props: Array of { date, rate } sorted chronologically.
 */
"use client";

import { CartesianGrid, Line, LineChart, XAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { format, parseISO } from "date-fns";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    rate: {
        label: "Attendance Rate",
        color: "#22c55e",
    },
    threshold: {
        label: "80% Threshold",
        color: "#ef4444",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface DailyAttendancePoint {
    date: string; // YYYY-MM-DD
    rate: number;
    isHoliday?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

interface DailyLineChartProps {
    data: DailyAttendancePoint[];
    showThreshold?: boolean;
}

export function DailyLineChart({ data, showThreshold = true }: DailyLineChartProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No daily attendance data available.
            </p>
        );
    }

    const chartData = data.map((d) => ({
        ...d,
        dateLabel: format(parseISO(d.date), "MMM d"),
    }));

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Daily Attendance Rate
                <GraphHelp>
                    Time-series line chart showing daily attendance rate with an optional 80%
                    threshold line. Helps identify trends and problem days.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <LineChart accessibilityLayer data={chartData}>
                    <CartesianGrid vertical={false} />
                    <XAxis
                        dataKey="dateLabel"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        interval="preserveStartEnd"
                        minTickGap={40}
                    />
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                indicator="dot"
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    {showThreshold && (
                        <ReferenceLine
                            y={80}
                            stroke="#ef4444"
                            strokeDasharray="4 4"
                            label={{
                                value: "80%",
                                position: "right",
                                fill: "#ef4444",
                                fontSize: 11,
                            }}
                        />
                    )}
                    <Line
                        dataKey="rate"
                        type="natural"
                        stroke="#22c55e"
                        strokeWidth={2}
                        dot={false}
                    />
                </LineChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

import { ReferenceLine } from "recharts";
