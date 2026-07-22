/**
 * SourceCompositionStackedBar — Stacked bar showing quantitative vs rubric contribution per subject.
 *
 * Visualisation: How much of each subject's score is from quantitative assessments vs rubric grades.
 * Props: Array of { subject, quantitativePct, rubricPct } entries.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

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
    quantitative: {
        label: "Quantitative",
        color: "#22c55e",
    },
    rubric: {
        label: "Rubric",
        color: "#f59e0b",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface SourceCompositionEntry {
    subject: string;
    quantitativePct: number;
    rubricPct: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface SourceCompositionStackedBarProps {
    data: SourceCompositionEntry[];
}

export function SourceCompositionStackedBar({ data }: SourceCompositionStackedBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No assessment source data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Assessment Source Composition
                <GraphHelp>
                    Stacked bar chart showing how much of each subject&rsquo;s score comes from
                    quantitative assessments versus rubric grades.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} layout="vertical" barCategoryGap="20%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" domain={[0, 100]} tickLine={false} axisLine={false} />
                    <YAxis
                        dataKey="subject"
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
                                indicator="dot"
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar
                        dataKey="quantitativePct"
                        stackId="a"
                        fill="#22c55e"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar dataKey="rubricPct" stackId="a" fill="#f59e0b" radius={[0, 4, 4, 0]} />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function SourceCompositionStackedBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
