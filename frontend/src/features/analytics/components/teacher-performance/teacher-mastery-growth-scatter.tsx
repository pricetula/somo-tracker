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

const chartConfig = {
    teacher: { label: "Teacher", color: "hsl(var(--chart-2))" },
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
            <p className="text-foreground text-sm font-medium">Mastery vs Growth</p>
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
                            style: { fontSize: 10, fill: "hsl(var(--muted-foreground))" },
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
                            style: { fontSize: 10, fill: "hsl(var(--muted-foreground))" },
                        }}
                    />
                    <ZAxis range={[80, 80]} />
                    <ReferenceLine y={0} stroke="hsl(var(--border))" />
                    <ChartTooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        content={<ChartTooltipContent />}
                    />
                    <Scatter data={data} fill="var(--color-teacher)" />
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
