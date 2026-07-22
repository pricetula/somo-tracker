/**
 * ClassVsGradeBar — Side-by-side bar: class rank vs grade rank comparison.
 *
 * Visualisation: Two rank values shown as bars.
 * Props: { classRank, classHeadcount, gradeRank, gradeHeadcount }.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    rank: {
        label: "Rank",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface ClassVsGradeData {
    classRank: number;
    classHeadcount: number;
    gradeRank: number;
    gradeHeadcount: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface ClassVsGradeBarProps {
    data: ClassVsGradeData;
}

export function ClassVsGradeBar({ data }: ClassVsGradeBarProps) {
    const chartData = [
        {
            label: "Class",
            rank: data.classRank,
            headcount: data.classHeadcount,
            percentile: (
                ((data.classHeadcount - data.classRank) / data.classHeadcount) *
                100
            ).toFixed(0),
        },
        {
            label: "Grade",
            rank: data.gradeRank,
            headcount: data.gradeHeadcount,
            percentile: (
                ((data.gradeHeadcount - data.gradeRank) / data.gradeHeadcount) *
                100
            ).toFixed(0),
        },
    ];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Class vs Grade Rank</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={chartData} barCategoryGap="40%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        label={{
                            value: "Rank",
                            angle: -90,
                            position: "left",
                            style: { fontSize: 10, fill: "hsl(var(--muted-foreground))" },
                        }}
                    />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(val, _name, _item) => {
                                    const value = Number(val);
                                    const payload = (_item as unknown as Record<string, unknown>)
                                        ?.payload as
                                        | { headcount: number; percentile: string }
                                        | undefined;
                                    if (payload)
                                        return `#${value} of ${payload.headcount} (top ${payload.percentile}%)`;
                                    return isNaN(value) ? val : `${value.toFixed(0)}`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="rank" radius={[4, 4, 0, 0]} barSize={60}>
                        {chartData.map((entry) => (
                            <Cell
                                key={entry.label}
                                fill={
                                    entry.label === "Class"
                                        ? "hsl(var(--chart-2))"
                                        : "hsl(var(--chart-4))"
                                }
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>

            <div className="flex items-center justify-center gap-6 text-xs">
                <div className="flex items-center gap-1">
                    <div className="h-3 w-3 rounded bg-[hsl(var(--chart-2))]" />
                    <span className="text-muted-foreground">
                        Class: #{data.classRank} of {data.classHeadcount}
                    </span>
                </div>
                <div className="flex items-center gap-1">
                    <div className="h-3 w-3 rounded bg-[hsl(var(--chart-4))]" />
                    <span className="text-muted-foreground">
                        Grade: #{data.gradeRank} of {data.gradeHeadcount}
                    </span>
                </div>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function ClassVsGradeBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
