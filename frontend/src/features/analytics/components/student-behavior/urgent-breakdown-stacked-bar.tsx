/**
 * UrgentBreakdownStackedBar — Urgent vs non-urgent within disciplinary and commendation.
 *
 * Visualisation: Stacked bar showing severity breakdown.
 * Props: Data grouped by type + urgency.
 */
"use client";

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartLegend,
    ChartLegendContent,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

const chartConfig = {
    urgent: { label: "Urgent", color: "hsl(var(--chart-1))" },
    nonUrgent: { label: "Non-Urgent", color: "hsl(var(--chart-3))" },
} satisfies ChartConfig;

interface Props {
    disciplinaryUrgent: number;
    disciplinaryNonUrgent: number;
    commendationUrgent: number;
    commendationNonUrgent: number;
}

export function UrgentBreakdownStackedBar({
    disciplinaryUrgent,
    disciplinaryNonUrgent,
    commendationUrgent,
    commendationNonUrgent,
}: Props) {
    const data = [
        {
            type: "Disciplinary",
            urgent: disciplinaryUrgent,
            nonUrgent: disciplinaryNonUrgent,
        },
        {
            type: "Commendation",
            urgent: commendationUrgent,
            nonUrgent: commendationNonUrgent,
        },
    ];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Urgency Breakdown</p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={data} barCategoryGap="40%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="type" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} allowDecimals={false} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar
                        dataKey="urgent"
                        stackId="a"
                        fill="var(--color-urgent)"
                        radius={[0, 0, 0, 0]}
                    />
                    <Bar
                        dataKey="nonUrgent"
                        stackId="a"
                        fill="var(--color-nonUrgent)"
                        radius={[0, 4, 4, 0]}
                    />
                </BarChart>
            </ChartContainer>
        </div>
    );
}

export function UrgentBreakdownStackedBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
