/**
 * SubjectComparisonBar — Bar chart comparing average % per learning area side-by-side.
 *
 * Visualisation: Each subject's average percentage with threshold colouring.
 * Props: Array of { learningAreaName, averagePercentage, level } entries.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubjectBarEntry {
    learningAreaName: string;
    averagePercentage: number;
    level?: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function levelColor(pct: number) {
    if (pct >= 80) return "hsl(var(--chart-2))"; // EE/ME
    if (pct >= 60) return "hsl(var(--chart-3))"; // AE
    return "hsl(var(--chart-1))"; // BE
}

// ─── Component ────────────────────────────────────────────────────────────

interface SubjectComparisonBarProps {
    data: SubjectBarEntry[];
}

export function SubjectComparisonBar({ data }: SubjectComparisonBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No subject comparison data available.
            </p>
        );
    }

    const chartConfig: ChartConfig = {};
    for (const entry of data) {
        chartConfig[entry.learningAreaName] = {
            label: entry.learningAreaName,
        };
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Subject Performance Comparison</p>
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
                        width={110}
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
                    <Bar dataKey="averagePercentage" radius={[0, 4, 4, 0]} barSize={20}>
                        {data.map((entry) => (
                            <Cell
                                key={entry.learningAreaName}
                                fill={levelColor(entry.averagePercentage)}
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>

            {/* Threshold legend */}
            <div className="flex items-center gap-4">
                <span className="text-muted-foreground text-xs">Levels:</span>
                {[
                    { label: "EE/ME (≥80%)", cls: "bg-[hsl(var(--chart-2))]" },
                    { label: "AE (60-79%)", cls: "bg-[hsl(var(--chart-3))]" },
                    { label: "BE (<60%)", cls: "bg-[hsl(var(--chart-1))]" },
                ].map((entry) => (
                    <div key={entry.label} className="flex items-center gap-1">
                        <div className={entry.cls + " h-3 w-3 rounded"} />
                        <span className="text-muted-foreground text-xs">{entry.label}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function SubjectComparisonBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
