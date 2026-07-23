/**
 * WorkloadComparisonBar — Horizontal bar comparing all teachers' workload.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

export interface TeacherWorkloadEntry {
    teacherName: string;
    periods: number;
    isOvercapacity: boolean;
    utilization: number;
}
interface Props {
    data: TeacherWorkloadEntry[];
}

export function WorkloadComparisonBar({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No workload data.</p>;
    const sorted = [...data].sort((a, b) => b.periods - a.periods);
    const chartConfig: ChartConfig = {};
    sorted.forEach((e) => {
        chartConfig[e.teacherName] = { label: e.teacherName };
    });

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Teacher Workload Comparison
                <GraphHelp>
                    Horizontal bar chart comparing weekly assigned periods across all teachers.
                    Green = within capacity, red = overcapacity.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/2] w-full">
                <BarChart data={sorted} layout="vertical" barCategoryGap="20%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" tickLine={false} axisLine={false} allowDecimals={false} />
                    <YAxis
                        dataKey="teacherName"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={120}
                        tick={{ fontSize: 11 }}
                    />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
                    <Bar dataKey="periods" radius={[0, 4, 4, 0]} barSize={18}>
                        {sorted.map((e) => (
                            <Cell
                                key={e.teacherName}
                                fill={e.isOvercapacity ? "#ef4444" : "#22c55e"}
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

export function WorkloadComparisonBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/2] w-full animate-pulse rounded" />
        </div>
    );
}
