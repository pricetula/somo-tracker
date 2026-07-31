/**
 * BeforeAfterComparison — Bar chart comparing mastery before and after remediation intervention.
 *
 * Visualisation: Paired bars showing improvement.
 * Props: Array of { subStrandName, beforePct, afterPct }.
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
    before: {
        label: "Before",
        color: "#ef4444",
    },
    after: {
        label: "After",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface BeforeAfterEntry {
    subStrandName: string;
    beforePct: number;
    afterPct: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface BeforeAfterComparisonProps {
    data: BeforeAfterEntry[];
}

export function BeforeAfterComparison({ data }: BeforeAfterComparisonProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No before/after data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Remediation Impact
                <GraphHelp>
                    Bar chart comparing mastery percentages before and after remediation
                    intervention, showing improvement for each sub-strand.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} barCategoryGap="30%">
                    <CartesianGrid vertical={false} />
                    <XAxis
                        dataKey="subStrandName"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        tick={{ fontSize: 11 }}
                    />
                    <YAxis domain={[0, 100]} tickLine={false} axisLine={false} />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                indicator="dot"
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(0)}%`;
                                }}
                            />
                        }
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar dataKey="beforePct" fill="#ef4444" radius={[4, 4, 0, 0]} />
                    <Bar dataKey="afterPct" fill="#22c55e" radius={[4, 4, 0, 0]} />
                </BarChart>
            </ChartContainer>

            {/* Improvement summary */}
            <div className="space-y-1 pt-1">
                {data.map((entry) => {
                    const diff = entry.afterPct - entry.beforePct;
                    return (
                        <div key={entry.subStrandName} className="flex items-center gap-2">
                            <span className="text-muted-foreground w-32 truncate text-xs">
                                {entry.subStrandName}
                            </span>
                            <span className="text-xs tabular-nums">
                                {entry.beforePct.toFixed(0)}% → {entry.afterPct.toFixed(0)}%
                            </span>
                            <span
                                className={
                                    diff > 0
                                        ? "text-xs font-medium text-emerald-600"
                                        : "text-destructive text-xs font-medium"
                                }
                            >
                                {diff > 0 ? "+" : ""}
                                {diff.toFixed(0)}%
                            </span>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
