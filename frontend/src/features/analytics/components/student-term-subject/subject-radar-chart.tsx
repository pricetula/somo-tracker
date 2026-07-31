/**
 * SubjectRadarChart — Radar / Spider chart showing performance across all subjects.
 *
 * Visualisation: Multi-axis radar — one axis per subject.
 * Props: Array of { subject, score } entries.
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
    score: {
        label: "Score",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubjectRadarEntry {
    subject: string;
    score: number;
    fullMark?: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface SubjectRadarChartProps {
    data: SubjectRadarEntry[];
}

export function SubjectRadarChartView({ data }: SubjectRadarChartProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No subject performance data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Performance Across Subjects
                <GraphHelp>
                    Multi-axis radar chart showing performance across all subjects. Each axis
                    represents a subject; the area reveals the student&rsquo;s relative strengths
                    and weaknesses.
                </GraphHelp>
            </p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[320px] w-full"
            >
                <RadarChart data={data}>
                    <PolarGrid stroke="#e5e7eb" />
                    <PolarAngleAxis dataKey="subject" tick={{ fontSize: 11, fill: "#6b7280" }} />
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
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Radar
                        name="Score"
                        dataKey="score"
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
