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
import { GraphHelp } from "@/features/analytics/components/graph-help";

const chartConfig = {
    urgent: { label: "Urgent", color: "#ef4444" },
    nonUrgent: { label: "Non-Urgent", color: "#3b82f6" },
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
            <p className="text-foreground text-sm font-medium">
                Urgency Breakdown
                <GraphHelp>
                    Stacked bar chart showing the breakdown of urgent versus non-urgent incidents
                    within disciplinary and commendation categories.
                </GraphHelp>
            </p>
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
                    <Bar dataKey="urgent" stackId="a" fill="#ef4444" radius={[0, 0, 0, 0]} />
                    <Bar dataKey="nonUrgent" stackId="a" fill="#3b82f6" radius={[0, 4, 4, 0]} />
                </BarChart>
            </ChartContainer>
        </div>
    );
}
