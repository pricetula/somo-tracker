/**
 * TeacherPerformanceRadar — Multi-axis radar of 5 teacher performance metrics.
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

const chartConfig = {
    value: { label: "Score", color: "#22c55e" },
} satisfies ChartConfig;

export interface TeacherMetricRadarData {
    metric: string;
    value: number;
}

interface Props {
    data: TeacherMetricRadarData[];
}

export function TeacherPerformanceRadar({ data }: Props) {
    if (!data.length)
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">No performance data.</p>
        );
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Performance Profile
                <GraphHelp>
                    Multi-axis radar chart showing five teacher performance metrics: subject mean
                    score, mastery rate, growth, timeliness, and coverage.
                </GraphHelp>
            </p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[300px] w-full"
            >
                <RadarChart data={data}>
                    <PolarGrid stroke="#e5e7eb" />
                    <PolarAngleAxis dataKey="metric" tick={{ fontSize: 10, fill: "#6b7280" }} />
                    <PolarRadiusAxis
                        angle={30}
                        domain={[0, 100]}
                        tick={{ fontSize: 10, fill: "#6b7280" }}
                    />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Radar
                        name="Score"
                        dataKey="value"
                        stroke="#22c55e"
                        fill="#22c55e"
                        fillOpacity={0.2}
                    />
                </RadarChart>
            </ChartContainer>
        </div>
    );
}
