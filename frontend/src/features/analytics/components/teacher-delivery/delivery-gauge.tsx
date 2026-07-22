/**
 * DeliveryGauge — on_time_submission_rate with target line (e.g. 95%).
 */
"use client";

import { Label, PolarGrid, PolarRadiusAxis, RadialBar, RadialBarChart } from "recharts";
import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

interface Props {
    rate: number;
    target?: number;
    teacherName?: string;
}

export function DeliveryGauge({ rate, target = 95, teacherName }: Props) {
    const color = rate >= target ? "#22c55e" : rate >= 80 ? "#3b82f6" : "#ef4444";
    const config: ChartConfig = { rate: { label: "Submission Rate", color } };
    const endAngle = 90 + (rate / 100) * 270;
    return (
        <div className="space-y-2">
            {teacherName && <p className="text-foreground text-sm font-medium">{teacherName}</p>}
            {!teacherName && (
                <p className="text-foreground text-sm font-medium">
                    On-Time Submission Rate
                    <GraphHelp>
                        Gauge chart showing the on-time lesson submission rate with a target line.
                        Green = on target, red = below threshold.
                    </GraphHelp>
                </p>
            )}
            <ChartContainer config={config} className="mx-auto aspect-square max-h-[220px] w-full">
                <RadialBarChart
                    data={[{ value: rate }]}
                    startAngle={90}
                    endAngle={endAngle}
                    innerRadius={70}
                    outerRadius={110}
                    barSize={16}
                >
                    <PolarGrid
                        gridType="circle"
                        radialLines={false}
                        stroke="#e5e7eb"
                        className="first:fill-muted last:fill-background"
                    />
                    <RadialBar dataKey="value" cornerRadius={8} />
                    <PolarRadiusAxis tick={false} tickLine={false} axisLine={false}>
                        <Label
                            content={({ viewBox }) => {
                                if (viewBox && "cx" in viewBox && "cy" in viewBox)
                                    return (
                                        <text
                                            x={viewBox.cx}
                                            y={viewBox.cy}
                                            textAnchor="middle"
                                            dominantBaseline="middle"
                                        >
                                            <tspan
                                                x={viewBox.cx}
                                                y={(viewBox.cy ?? 0) - 8}
                                                className="fill-foreground text-3xl font-bold"
                                            >
                                                {rate.toFixed(1)}%
                                            </tspan>
                                            <tspan
                                                x={viewBox.cx}
                                                y={(viewBox.cy ?? 0) + 16}
                                                className="fill-muted-foreground text-xs"
                                            >
                                                {rate >= target
                                                    ? "On Target"
                                                    : `Target: ${target}%`}
                                            </tspan>
                                        </text>
                                    );
                                return null;
                            }}
                        />
                    </PolarRadiusAxis>
                </RadialBarChart>
            </ChartContainer>
        </div>
    );
}

export function DeliveryGaugeSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[220px] w-full animate-pulse rounded-full" />
        </div>
    );
}
