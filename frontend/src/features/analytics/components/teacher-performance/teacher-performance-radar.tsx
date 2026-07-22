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

const chartConfig = {
    value: { label: "Score", color: "hsl(var(--chart-2))" },
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
            <p className="text-foreground text-sm font-medium">Performance Profile</p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[300px] w-full"
            >
                <RadarChart data={data}>
                    <PolarGrid stroke="hsl(var(--border))" />
                    <PolarAngleAxis
                        dataKey="metric"
                        tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                    />
                    <PolarRadiusAxis
                        angle={30}
                        domain={[0, 100]}
                        tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                    />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Radar
                        name="Score"
                        dataKey="value"
                        stroke="var(--color-value)"
                        fill="var(--color-value)"
                        fillOpacity={0.2}
                    />
                </RadarChart>
            </ChartContainer>
        </div>
    );
}

export function TeacherPerformanceRadarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[300px] w-full animate-pulse rounded" />
        </div>
    );
}
