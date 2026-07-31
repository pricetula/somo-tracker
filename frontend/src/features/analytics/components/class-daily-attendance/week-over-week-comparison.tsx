/**
 * WeekOverWeekComparison — Bar chart comparing current week vs previous week vs same week last term.
 *
 * Visualisation: Three-bar group for each metric (attendance rate, absent count, etc.).
 * Props: Array of weekly comparison data.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

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
    currentWeek: {
        label: "Current Week",
        color: "#22c55e",
    },
    previousWeek: {
        label: "Previous Week",
        color: "#3b82f6",
    },
    sameWeekLastTerm: {
        label: "Same Week Last Term",
        color: "#8b5cf6",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface WeekOverWeekData {
    metric: string;
    currentWeek: number;
    previousWeek: number;
    sameWeekLastTerm?: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface WeekOverWeekComparisonProps {
    data: WeekOverWeekData[];
}

export function WeekOverWeekComparison({ data }: WeekOverWeekComparisonProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No week-over-week data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Week-over-Week Comparison
                <GraphHelp>
                    Bar chart comparing current week against the previous week and the same week
                    last term to track attendance trends.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} barCategoryGap="30%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="metric" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar dataKey="currentWeek" fill="#22c55e" radius={[4, 4, 0, 0]} />
                    <Bar dataKey="previousWeek" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                    {data[0]?.sameWeekLastTerm !== undefined && (
                        <Bar dataKey="sameWeekLastTerm" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
                    )}
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
