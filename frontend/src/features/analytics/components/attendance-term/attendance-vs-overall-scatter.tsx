/**
 * AttendanceVsOverallScatter — Scatter plot: Attendance % vs Overall Mean %.
 *
 * Visualisation: Correlation check between attendance and academic performance.
 * Props: Array of { attendancePercentage, overallMeanPercentage, studentName } data points.
 */
"use client";

import { CartesianGrid, Scatter, ScatterChart, XAxis, YAxis, ZAxis } from "recharts";

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
        label: "Attendance %",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface AttendanceVsPerformancePoint {
    studentName: string;
    attendancePercentage: number;
    overallMeanPercentage: number;
    studentId?: string;
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceVsOverallScatterProps {
    data: AttendanceVsPerformancePoint[];
}

export function AttendanceVsOverallScatter({ data }: AttendanceVsOverallScatterProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No comparison data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Attendance vs Academic Performance
                <GraphHelp>
                    Scatter plot showing the correlation between attendance percentage and overall
                    academic performance. Each dot represents a student.
                </GraphHelp>
            </p>
            <p className="text-muted-foreground text-xs">
                Each dot = a student. Correlation between attendance and overall score.
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/2] w-full">
                <ScatterChart margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                        dataKey="attendancePercentage"
                        name="Attendance %"
                        type="number"
                        domain={[0, 100]}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        label={{
                            value: "Attendance %",
                            position: "bottom",
                            offset: -4,
                            style: { fontSize: 11, fill: "#6b7280" },
                        }}
                    />
                    <YAxis
                        dataKey="overallMeanPercentage"
                        name="Overall Mean %"
                        type="number"
                        domain={[0, 100]}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        label={{
                            value: "Overall Mean %",
                            angle: -90,
                            position: "left",
                            style: { fontSize: 11, fill: "#6b7280" },
                        }}
                    />
                    <ZAxis range={[64, 64]} />
                    <ChartTooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        content={
                            <ChartTooltipContent
                                indicator="dot"
                                formatter={(val: unknown, name) => {
                                    const value = Number(val);
                                    if (isNaN(value)) return "";
                                    if (
                                        name === "Attendance %" ||
                                        name === "attendancePercentage"
                                    ) {
                                        return `${value.toFixed(1)}%`;
                                    }
                                    if (
                                        name === "Overall Mean %" ||
                                        name === "overallMeanPercentage"
                                    ) {
                                        return `${value.toFixed(1)}%`;
                                    }
                                    return value;
                                }}
                            />
                        }
                    />
                    <Scatter name="Students" data={data} fill="#22c55e" />
                </ScatterChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function AttendanceVsOverallScatterSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/2] w-full animate-pulse rounded" />
        </div>
    );
}
