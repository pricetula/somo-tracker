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
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    mastery: {
        label: "Mastery",
        color: "#22c55e",
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
            <p className="text-foreground text-sm font-medium">
                Skill Profile
                <GraphHelp>
                    Multi-axis radar chart comparing mastery across different sub-strands at a
                    glance, helping identify strengths and areas needing improvement.
                </GraphHelp>
            </p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[320px] w-full"
            >
                <RadarChart data={data}>
                    <PolarGrid stroke="#e5e7eb" />
                    <PolarAngleAxis
                        dataKey="subStrandName"
                        tick={{ fontSize: 10, fill: "#6b7280" }}
                    />
                    <PolarRadiusAxis
                        angle={30}
                        domain={[0, 100]}
                        tick={{ fontSize: 10, fill: "#6b7280" }}
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
                        stroke="#22c55e"
                        fill="#22c55e"
                        fillOpacity={0.2}
                    />
                </RadarChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
