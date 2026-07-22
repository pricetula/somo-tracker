/**
 * LevelDonutChart — Donut chart showing proportion of subjects at each performance level.
 *
 * Visualisation: EE/ME/AE/BE distribution as a donut.
 * Props: { exceedingCount, meetingCount, approachingCount, belowCount }.
 */
"use client";

import { Pie, PieChart } from "recharts";

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
    exceeding: {
        label: "Exceeding (EE)",
        color: "hsl(var(--chart-2))",
    },
    meeting: {
        label: "Meeting (ME)",
        color: "hsl(var(--chart-3))",
    },
    approaching: {
        label: "Approaching (AE)",
        color: "hsl(var(--chart-4))",
    },
    below: {
        label: "Below (BE)",
        color: "hsl(var(--chart-1))",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface LevelDistribution {
    exceeding: number;
    meeting: number;
    approaching: number;
    below: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface LevelDonutChartProps {
    data: LevelDistribution;
}

export function LevelDonutChart({ data }: LevelDonutChartProps) {
    const total = data.exceeding + data.meeting + data.approaching + data.below;

    if (total === 0) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No level data available.
            </p>
        );
    }

    const pieData = [
        { name: "exceeding", value: data.exceeding, fill: "var(--color-exceeding)" },
        { name: "meeting", value: data.meeting, fill: "var(--color-meeting)" },
        { name: "approaching", value: data.approaching, fill: "var(--color-approaching)" },
        { name: "below", value: data.below, fill: "var(--color-below)" },
    ].filter((d) => d.value > 0);

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Performance Level Distribution</p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[260px] w-full"
            >
                <PieChart>
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                formatter={(val) => {
                                    const v = Number(val);
                                    if (isNaN(v)) return "";
                                    const pct = total > 0 ? ((v / total) * 100).toFixed(0) : "0";
                                    return `${v} subjects (${pct}%)`;
                                }}
                            />
                        }
                    />
                    <Pie
                        data={pieData}
                        dataKey="value"
                        nameKey="name"
                        innerRadius={60}
                        outerRadius={100}
                        strokeWidth={2}
                        stroke="hsl(var(--background))"
                        paddingAngle={2}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                </PieChart>
            </ChartContainer>
            <p className="text-muted-foreground text-center text-xs">
                {total} subject{total !== 1 ? "s" : ""} total
            </p>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function LevelDonutChartSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[260px] w-full animate-pulse rounded-full" />
        </div>
    );
}
