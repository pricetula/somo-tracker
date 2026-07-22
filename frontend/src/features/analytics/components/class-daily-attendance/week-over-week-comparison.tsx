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

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    currentWeek: {
        label: "Current Week",
        color: "hsl(var(--chart-2))",
    },
    previousWeek: {
        label: "Previous Week",
        color: "hsl(var(--chart-3))",
    },
    sameWeekLastTerm: {
        label: "Same Week Last Term",
        color: "hsl(var(--chart-5))",
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
            <p className="text-foreground text-sm font-medium">Week-over-Week Comparison</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={data} barCategoryGap="30%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="metric" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar
                        dataKey="currentWeek"
                        fill="var(--color-currentWeek)"
                        radius={[4, 4, 0, 0]}
                    />
                    <Bar
                        dataKey="previousWeek"
                        fill="var(--color-previousWeek)"
                        radius={[4, 4, 0, 0]}
                    />
                    {data[0]?.sameWeekLastTerm !== undefined && (
                        <Bar
                            dataKey="sameWeekLastTerm"
                            fill="var(--color-sameWeekLastTerm)"
                            radius={[4, 4, 0, 0]}
                        />
                    )}
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function WeekOverWeekComparisonSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
