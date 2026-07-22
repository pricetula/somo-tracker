/**
 * StrandMasteryBar — Green/red bars showing mastery % per sub-strand.
 *
 * Visualisation: Horizontal bars with threshold colouring.
 * Props: Array of { subStrandName, masteryPercentage, requiresRemediation }.
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

export interface MasteryBarEntry {
    subStrandName: string;
    masteryPercentage: number;
    requiresRemediation: boolean;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function barFill(pct: number): string {
    if (pct >= 80) return "hsl(var(--chart-2))";
    if (pct >= 60) return "hsl(var(--chart-3))";
    if (pct >= 40) return "hsl(var(--chart-4))";
    return "hsl(var(--chart-1))";
}

// ─── Component ────────────────────────────────────────────────────────────

interface StrandMasteryBarProps {
    data: MasteryBarEntry[];
}

export function StrandMasteryBar({ data }: StrandMasteryBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No mastery data available.
            </p>
        );
    }

    // Sort by mastery ascending
    const sorted = [...data].sort((a, b) => a.masteryPercentage - b.masteryPercentage);

    const chartConfig: ChartConfig = {};
    for (const entry of data) {
        chartConfig[entry.subStrandName] = {
            label: entry.subStrandName,
        };
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Mastery by Sub-Strand</p>
            <ChartContainer config={chartConfig} className="aspect-[3/2] w-full">
                <BarChart data={sorted} layout="vertical" barCategoryGap="20%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" domain={[0, 100]} tickLine={false} axisLine={false} />
                    <YAxis
                        dataKey="subStrandName"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={130}
                        tick={{ fontSize: 11 }}
                    />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(0)}%`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="masteryPercentage" radius={[0, 4, 4, 0]} barSize={16}>
                        {sorted.map((entry) => (
                            <Cell
                                key={entry.subStrandName}
                                fill={barFill(entry.masteryPercentage)}
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function StrandMasteryBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/2] w-full animate-pulse rounded" />
        </div>
    );
}
