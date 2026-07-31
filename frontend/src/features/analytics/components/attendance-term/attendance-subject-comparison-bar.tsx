/**
 * AttendanceSubjectComparisonBar — Bar chart comparing attendance % by learning area.
 *
 * Visualisation: Reveals subject-specific truancy patterns.
 * Props: Array of { learningAreaName, percentage } entries.
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

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubjectAttendanceEntry {
    learningAreaName: string;
    percentage: number;
    color?: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function barColor(pct: number) {
    if (pct >= 90) return "#22c55e";
    if (pct >= 75) return "#3b82f6";
    if (pct >= 50) return "#f59e0b";
    return "#ef4444";
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceSubjectComparisonBarProps {
    data: SubjectAttendanceEntry[];
}

export function AttendanceSubjectComparisonBar({ data }: AttendanceSubjectComparisonBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No subject attendance data available.
            </p>
        );
    }

    const chartConfig: ChartConfig = {};
    for (const entry of data) {
        chartConfig[entry.learningAreaName] = {
            label: entry.learningAreaName,
            color: entry.color ?? barColor(entry.percentage),
        };
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Attendance by Learning Area
                <GraphHelp>
                    Bar chart comparing attendance percentages by learning area, revealing
                    subject-specific truancy patterns.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} layout="vertical">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" domain={[0, 100]} tickLine={false} axisLine={false} />
                    <YAxis
                        dataKey="learningAreaName"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={100}
                    />
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
                    <Bar dataKey="percentage" radius={[0, 4, 4, 0]} barSize={20}>
                        {data.map((entry) => (
                            <Cell
                                key={entry.learningAreaName}
                                fill={entry.color ?? barColor(entry.percentage)}
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

// Import Cell from recharts for per-bar coloring
import { Cell, YAxis } from "recharts";
