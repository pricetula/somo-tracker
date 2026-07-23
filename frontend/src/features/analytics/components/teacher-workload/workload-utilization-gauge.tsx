/**
 * WorkloadUtilizationGauge — Utilization % with overcapacity threshold line.
 */
"use client";

import { Label, PolarGrid, PolarRadiusAxis, RadialBar, RadialBarChart } from "recharts";
import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

interface Props {
    utilization: number;
    isOvercapacity: boolean;
    teacherName?: string;
}

export function WorkloadUtilizationGauge({ utilization, isOvercapacity, teacherName }: Props) {
    const color = isOvercapacity ? "#ef4444" : "#22c55e";
    const config: ChartConfig = { utilization: { label: "Utilization", color } };
    const endAngle = 90 + (utilization / 100) * 270;
    return (
        <div className="space-y-2">
            {teacherName && <p className="text-foreground text-sm font-medium">{teacherName}</p>}
            {!teacherName && (
                <p className="text-foreground text-sm font-medium">
                    Workload Utilization
                    <GraphHelp>
                        Gauge chart showing teacher workload utilisation percentage with an
                        overcapacity threshold. Green = within capacity, red = overcapacity.
                    </GraphHelp>
                </p>
            )}
            <ChartContainer config={config} className="mx-auto aspect-square max-h-[220px] w-full">
                <RadialBarChart
                    data={[{ value: utilization }]}
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
                                                {utilization.toFixed(1)}%
                                            </tspan>
                                            <tspan
                                                x={viewBox.cx}
                                                y={(viewBox.cy ?? 0) + 16}
                                                className={
                                                    isOvercapacity
                                                        ? "fill-destructive text-xs"
                                                        : "fill-muted-foreground text-xs"
                                                }
                                            >
                                                {isOvercapacity ? "Overcapacity" : "Within Range"}
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

export function WorkloadUtilizationGaugeSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[220px] w-full animate-pulse rounded-full" />
        </div>
    );
}
