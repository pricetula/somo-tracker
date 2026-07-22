/**
 * DeliveryWeeklyStackedBar — Marked vs Missed vs Unaccounted per week.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartLegend,
    ChartLegendContent,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

const chartConfig = {
    marked: { label: "Marked", color: "hsl(var(--chart-2))" },
    missed: { label: "Missed", color: "hsl(var(--chart-1))" },
    unaccounted: { label: "Unaccounted", color: "hsl(var(--chart-4))" },
} satisfies ChartConfig;

export interface WeeklyDeliveryRow {
    weekLabel: string;
    marked: number;
    missed: number;
    unaccounted: number;
}
interface Props {
    data: WeeklyDeliveryRow[];
}

export function DeliveryWeeklyStackedBar({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No data.</p>;
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Weekly Delivery Breakdown</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={data} barCategoryGap="20%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="weekLabel" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar
                        dataKey="marked"
                        stackId="a"
                        fill="var(--color-marked)"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar
                        dataKey="missed"
                        stackId="a"
                        fill="var(--color-missed)"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar
                        dataKey="unaccounted"
                        stackId="a"
                        fill="var(--color-unaccounted)"
                        radius={[0, 4, 4, 0]}
                    />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

export function DeliveryWeeklyStackedBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
