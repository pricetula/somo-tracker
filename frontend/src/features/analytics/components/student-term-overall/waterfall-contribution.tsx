/**
 * WaterfallContribution — Waterfall chart showing how each subject contributes to (or drags down) the overall.
 *
 * Visualisation: Starting from overall mean, each subject shows its deviation.
 * Props: { overallMean, subjects: [{ name, score }] }.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";

import { GraphHelp } from "@/components/GraphHelp";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubjectContribution {
    name: string;
    contribution: number; // deviation from overall mean (positive = above, negative = below)
    score: number;
    overallMean: number;
}

// ─── Waterfall data builder ──────────────────────────────────────────────

interface WaterfallBar {
    label: string;
    start: number;
    end: number;
    isTotal: boolean;
    isPositive: boolean;
}

function buildWaterfallBars(
    overallMean: number,
    subjects: { name: string; score: number }[]
): WaterfallBar[] {
    const bars: WaterfallBar[] = [];
    let runningTotal = overallMean;

    // Starting bar
    bars.push({
        label: "Overall",
        start: 0,
        end: overallMean,
        isTotal: true,
        isPositive: true,
    });

    // Each subject's contribution
    for (const sub of subjects) {
        const diff = sub.score - overallMean;
        const newTotal = runningTotal + diff;
        bars.push({
            label: sub.name,
            start: runningTotal,
            end: newTotal,
            isTotal: false,
            isPositive: diff >= 0,
        });
        runningTotal = newTotal;
    }

    // Final total
    bars.push({
        label: "Final",
        start: 0,
        end: runningTotal,
        isTotal: true,
        isPositive: runningTotal >= overallMean,
    });

    return bars;
}

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    start: { label: "Start" },
    end: { label: "End" },
} satisfies ChartConfig;

// ─── Component ────────────────────────────────────────────────────────────

interface WaterfallContributionProps {
    overallMean: number;
    subjects: { name: string; score: number }[];
}

export function WaterfallContribution({ overallMean, subjects }: WaterfallContributionProps) {
    if (!subjects.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No waterfall data available.
            </p>
        );
    }

    const bars = buildWaterfallBars(overallMean, subjects);

    // Chart data: each bar needs an invisible "base" and visible "change"
    const chartData = bars.map((b) => ({
        label: b.label,
        base: b.isTotal ? 0 : Math.min(b.start, b.end),
        change: Math.abs(b.end - b.start),
        total: b.end,
        isTotal: b.isTotal,
        isPositive: b.isPositive,
    }));

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Subject Contribution (Waterfall)
                <GraphHelp>
                    Waterfall chart showing how each subject contributes to or drags down the
                    overall mean score. Green bars = above average, red = below.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={chartData} barCategoryGap="20%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(val, name) => {
                                    const v = Number(val);
                                    if (isNaN(v)) return "";
                                    if (name === "base") return null;
                                    return `${v.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    {/* Invisible base bar */}
                    <Bar dataKey="base" stackId="a" fill="transparent" />
                    {/* Visible change bar */}
                    <Bar dataKey="change" stackId="a" radius={[0, 0, 0, 0]}>
                        {chartData.map((entry, i) => (
                            <Cell
                                key={i}
                                fill={
                                    entry.isTotal
                                        ? "#6b7280"
                                        : entry.isPositive
                                          ? "#22c55e"
                                          : "#ef4444"
                                }
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>

            <div className="flex items-center gap-4">
                <span className="text-muted-foreground text-xs">Contribution:</span>
                <div className="flex items-center gap-1">
                    <div className="h-3 w-3 rounded bg-[#22c55e]" />
                    <span className="text-muted-foreground text-xs">Positive</span>
                </div>
                <div className="flex items-center gap-1">
                    <div className="h-3 w-3 rounded bg-[#ef4444]" />
                    <span className="text-muted-foreground text-xs">Negative</span>
                </div>
                <div className="flex items-center gap-1">
                    <div className="bg-muted-foreground h-3 w-3 rounded" />
                    <span className="text-muted-foreground text-xs">Total</span>
                </div>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function WaterfallContributionSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
