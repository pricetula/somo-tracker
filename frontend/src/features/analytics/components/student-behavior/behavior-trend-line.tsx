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

const chartConfig = {
    commendations: { label: "Commendations", color: "hsl(var(--chart-2))" },
    disciplinary: { label: "Disciplinary", color: "hsl(var(--chart-1))" },
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
            <p className="text-foreground text-sm font-medium">Behaviour Trend Across Terms</p>
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
                        stroke="var(--color-commendations)"
                        strokeWidth={2}
                        dot={false}
                    />
                    <Line
                        type="monotone"
                        dataKey="disciplinary"
                        stroke="var(--color-disciplinary)"
                        strokeWidth={2}
                        dot={false}
                    />
                </LineChart>
            </ChartContainer>
        </div>
    );
}

export function BehaviorTrendLineSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
