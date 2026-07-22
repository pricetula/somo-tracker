/**
 * DeliveryComparisonBar — Side-by-side delivery rates across teaching staff.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

export interface TeacherDeliveryRate {
    teacherName: string;
    submissionRate: number;
}
interface Props {
    data: TeacherDeliveryRate[];
}

export function DeliveryComparisonBar({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No data.</p>;
    const sorted = [...data].sort((a, b) => b.submissionRate - a.submissionRate);
    const chartConfig: ChartConfig = {};
    sorted.forEach((e) => {
        chartConfig[e.teacherName] = { label: e.teacherName };
    });

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Teacher Delivery Rates</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={sorted} layout="vertical" barCategoryGap="20%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" domain={[0, 100]} tickLine={false} axisLine={false} />
                    <YAxis
                        dataKey="teacherName"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={120}
                        tick={{ fontSize: 11 }}
                    />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(val: unknown) => {
                                    const v = Number(val);
                                    return isNaN(v) ? "" : `${v.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="submissionRate" radius={[0, 4, 4, 0]} barSize={18}>
                        {sorted.map((e) => (
                            <Cell
                                key={e.teacherName}
                                fill={
                                    e.submissionRate >= 95
                                        ? "hsl(var(--chart-2))"
                                        : e.submissionRate >= 80
                                          ? "hsl(var(--chart-3))"
                                          : "hsl(var(--chart-1))"
                                }
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

export function DeliveryComparisonBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
