/**
 * TeacherMasteryGrowthScatter — Plots each teacher: mastery rate vs growth rate.
 */
"use client";

import { CartesianGrid, ReferenceLine, Scatter, ScatterChart, XAxis, YAxis, ZAxis } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

const chartConfig = {
    teacher: { label: "Teacher", color: "#22c55e" },
} satisfies ChartConfig;

export interface MasteryVsGrowthPoint {
    teacherName: string;
    masteryRate: number;
    growthRate: number;
}

interface Props {
    data: MasteryVsGrowthPoint[];
}

export function TeacherMasteryGrowthScatter({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No data.</p>;
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Mastery vs Growth
                <GraphHelp>
                    Scatter plot plotting each teacher&rsquo;s mastery rate against their growth
                    rate. Identifies high-growth, high-mastery standouts.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[4/3] w-full">
                <ScatterChart margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                        dataKey="masteryRate"
                        name="Mastery Rate"
                        type="number"
                        domain={[0, 100]}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        label={{
                            value: "Mastery %",
                            position: "bottom",
                            offset: -4,
                            style: { fontSize: 10, fill: "#6b7280" },
                        }}
                    />
                    <YAxis
                        dataKey="growthRate"
                        name="Growth Rate"
                        type="number"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        label={{
                            value: "Growth (pts)",
                            angle: -90,
                            position: "left",
                            style: { fontSize: 10, fill: "#6b7280" },
                        }}
                    />
                    <ZAxis range={[80, 80]} />
                    <ReferenceLine y={0} stroke="#e5e7eb" />
                    <ChartTooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        content={<ChartTooltipContent />}
                    />
                    <Scatter data={data} fill="#22c55e" />
                </ScatterChart>
            </ChartContainer>
        </div>
    );
}

export function TeacherMasteryGrowthScatterSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted aspect-[4/3] w-full animate-pulse rounded" />
        </div>
    );
}
