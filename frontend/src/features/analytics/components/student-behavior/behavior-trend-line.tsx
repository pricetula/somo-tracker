/**
 * BehaviorTrendLine — Commendation and disciplinary counts across terms.
 * Props: Array of { termName, commendations, disciplinary }.
 */
"use client";

import { CartesianGrid, Line, LineChart, XAxis } from "recharts";
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
    commendations: { label: "Commendations", color: "#22c55e" },
    disciplinary: { label: "Disciplinary", color: "#ef4444" },
} satisfies ChartConfig;

export interface BehaviorTrendPoint {
    termName: string;
    commendations: number;
    disciplinary: number;
}

interface Props {
    data: BehaviorTrendPoint[];
}

export function BehaviorTrendLine({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No trend data.</p>;
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Behaviour Trend Across Terms
                <GraphHelp>
                    Line chart showing commendation and disciplinary counts across terms to track
                    behaviour patterns over time.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <LineChart data={data}>
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="termName" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent indicator="dot" />}
                    />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Line
                        type="monotone"
                        dataKey="commendations"
                        stroke="#22c55e"
                        strokeWidth={2}
                        dot={false}
                    />
                    <Line
                        type="monotone"
                        dataKey="disciplinary"
                        stroke="#ef4444"
                        strokeWidth={2}
                        dot={false}
                    />
                </LineChart>
            </ChartContainer>
        </div>
    );
}
