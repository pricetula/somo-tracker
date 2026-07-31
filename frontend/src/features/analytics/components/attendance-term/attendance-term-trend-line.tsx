/**
 * AttendanceTermTrendLine — Line chart showing attendance trend across terms.
 *
 * Visualisation: Attendance trend across multiple terms.
 * Props: Array of { termName, percentage } points.
 */
"use client";

import { CartesianGrid, Dot, Line, LineChart, XAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    attendance: {
        label: "Attendance",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface TermAttendancePoint {
    termName: string;
    percentage: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceTermTrendLineProps {
    data: TermAttendancePoint[];
}

export function AttendanceTermTrendLine({ data }: AttendanceTermTrendLineProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No attendance trend data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Attendance Trend Across Terms
                <GraphHelp>
                    Line chart displaying attendance trends across multiple terms, helping identify
                    patterns of improvement or decline over time.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <LineChart
                    accessibilityLayer
                    data={data}
                    margin={{ top: 8, left: 8, right: 8, bottom: 8 }}
                >
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="termName" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
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
                    <Line
                        dataKey="percentage"
                        type="natural"
                        stroke="#22c55e"
                        strokeWidth={2}
                        dot={({ payload }) => (
                            <Dot key={payload.termName} r={5} fill="#22c55e" stroke="#22c55e" />
                        )}
                    />
                </LineChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
