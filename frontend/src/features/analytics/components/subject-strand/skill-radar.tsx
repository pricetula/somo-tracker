/**
 * SkillRadar — Multi-axis radar chart comparing sub-strand mastery.
 *
 * Visualisation: Radar with one axis per sub-strand.
 * Props: Array of { subStrandName, masteryPercentage }.
 */
"use client";

import { PolarAngleAxis, PolarGrid, PolarRadiusAxis, Radar, RadarChart } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    mastery: {
        label: "Mastery",
        color: "hsl(var(--chart-2))",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface SkillRadarEntry {
    subStrandName: string;
    masteryPercentage: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface SkillRadarProps {
    data: SkillRadarEntry[];
}

export function SkillRadar({ data }: SkillRadarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No skill profile data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Skill Profile</p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[320px] w-full"
            >
                <RadarChart data={data}>
                    <PolarGrid stroke="hsl(var(--border))" />
                    <PolarAngleAxis
                        dataKey="subStrandName"
                        tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                    />
                    <PolarRadiusAxis
                        angle={30}
                        domain={[0, 100]}
                        tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                    />
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(0)}%`;
                                }}
                            />
                        }
                    />
                    <Radar
                        name="Mastery"
                        dataKey="masteryPercentage"
                        stroke="var(--color-mastery)"
                        fill="var(--color-mastery)"
                        fillOpacity={0.2}
                    />
                </RadarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function SkillRadarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-32 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[320px] w-full animate-pulse rounded" />
        </div>
    );
}
