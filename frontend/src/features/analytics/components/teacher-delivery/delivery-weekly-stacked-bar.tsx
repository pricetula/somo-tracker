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
import { GraphHelp } from "@/features/analytics/components/graph-help";

const chartConfig = {
    marked: { label: "Marked", color: "#22c55e" },
    missed: { label: "Missed", color: "#ef4444" },
    unaccounted: { label: "Unaccounted", color: "#f59e0b" },
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
            <p className="text-foreground text-sm font-medium">
                Weekly Delivery Breakdown
                <GraphHelp>
                    Stacked bar chart showing weekly lesson delivery breakdown: Marked, Missed, and
                    Unaccounted.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={data} barCategoryGap="20%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="weekLabel" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar dataKey="marked" stackId="a" fill="#22c55e" radius={[0, 0, 0, 0]} />
                    <Bar dataKey="missed" stackId="a" fill="#ef4444" radius={[0, 0, 0, 0]} />
                    <Bar dataKey="unaccounted" stackId="a" fill="#f59e0b" radius={[0, 4, 4, 0]} />
                </BarChart>
            </ChartContainer>
        </div>
    );
}
