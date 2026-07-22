/**
 * LevelPieChart — Pie chart showing proportion of indicators at each level for a sub-strand.
 *
 * Visualisation: EE/ME/AE/BE distribution for one sub-strand.
 * Props: { exceedingCount, meetingCount, approachingCount, belowCount, subStrandName }.
 */
"use client";

import { Pie, PieChart } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
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

export interface LevelPieData {
    subStrandName: string;
    exceedingCount: number;
    meetingCount: number;
    approachingCount: number;
    belowCount: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface LevelPieChartProps {
    data: LevelPieData;
}

export function LevelPieChartView({ data }: LevelPieChartProps) {
    const total = data.exceedingCount + data.meetingCount + data.approachingCount + data.belowCount;

    if (total === 0) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No indicator data available.
            </p>
        );
    }

    const pieData = [
        {
            name: "exceeding",
            value: data.exceedingCount,
            fill: "var(--color-exceeding)",
        },
        {
            name: "meeting",
            value: data.meetingCount,
            fill: "var(--color-meeting)",
        },
        {
            name: "approaching",
            value: data.approachingCount,
            fill: "var(--color-approaching)",
        },
        {
            name: "below",
            value: data.belowCount,
            fill: "var(--color-below)",
        },
    ].filter((d) => d.value > 0);

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Indicator Distribution — {data.subStrandName}
            </p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[240px] w-full"
            >
                <PieChart>
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value} indicator${value !== 1 ? "s" : ""}`;
                                }}
                            />
                        }
                    />
                    <Pie
                        data={pieData}
                        dataKey="value"
                        nameKey="name"
                        innerRadius={50}
                        outerRadius={90}
                        strokeWidth={2}
                        stroke="hsl(var(--background))"
                        paddingAngle={2}
                    />
                </PieChart>
            </ChartContainer>
            <p className="text-muted-foreground text-center text-xs">
                {total} indicator{total !== 1 ? "s" : ""} total
            </p>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function LevelPieChartSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[240px] w-full animate-pulse rounded-full" />
        </div>
    );
}
