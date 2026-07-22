/**
 * TermOverTermComparison — Side-by-side bar chart comparing across terms.
 *
 * Visualisation: Term 1 vs Term 2 vs Term 3 overall scores.
 * Props: Array of { termName, score } entries.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    score: {
        label: "Overall %",
        color: "hsl(var(--chart-2))",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface TermComparisonEntry {
    termName: string;
    score: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface TermOverTermComparisonProps {
    data: TermComparisonEntry[];
}

export function TermOverTermComparison({ data }: TermOverTermComparisonProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No term comparison data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Term-over-Term Comparison</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} barCategoryGap="30%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="termName" tickLine={false} axisLine={false} tickMargin={8} />
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
                    <Bar
                        dataKey="score"
                        fill="var(--color-score)"
                        radius={[4, 4, 0, 0]}
                        barSize={50}
                    />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function TermOverTermComparisonSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
