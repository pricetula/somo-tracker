/**
 * LevelDistributionBar — Horizontal bar chart showing count of subjects per performance level.
 *
 * Visualisation: "At a glance" report card — how many subjects at EE, ME, AE, BE.
 * Props: { exceedingCount, meetingCount, approachingCount, belowCount }.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Types ────────────────────────────────────────────────────────────────

export interface LevelCountData {
    level: string;
    count: number;
}

// ─── Config ───────────────────────────────────────────────────────────────

function buildConfig(data: LevelCountData[]): ChartConfig {
    const config: ChartConfig = {};
    for (const entry of data) {
        config[entry.level] = {
            label: entry.level,
        };
    }
    return config;
}

function levelColor(level: string): string {
    switch (level) {
        case "Exceeding (EE)":
        case "EE":
            return "hsl(var(--chart-2))";
        case "Meeting (ME)":
        case "ME":
            return "hsl(var(--chart-3))";
        case "Approaching (AE)":
        case "AE":
            return "hsl(var(--chart-4))";
        case "Below (BE)":
        case "BE":
            return "hsl(var(--chart-1))";
        default:
            return "hsl(var(--muted))";
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface LevelDistributionBarProps {
    data: LevelCountData[];
}

export function LevelDistributionBar({ data }: LevelDistributionBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No level distribution data available.
            </p>
        );
    }

    const chartConfig = buildConfig(data);
    const ordered = ["EE", "ME", "AE", "BE"]
        .map((l) => data.find((d) => d.level === l))
        .filter(Boolean) as LevelCountData[];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Subjects per Performance Level</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart accessibilityLayer data={ordered} layout="vertical" barCategoryGap="30%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" tickLine={false} axisLine={false} allowDecimals={false} />
                    <YAxis
                        dataKey="level"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={40}
                    />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value} subject${value !== 1 ? "s" : ""}`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="count" radius={[0, 4, 4, 0]} barSize={28}>
                        {ordered.map((entry) => (
                            <Cell key={entry.level} fill={levelColor(entry.level)} />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function LevelDistributionBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}

import { Cell } from "recharts";
