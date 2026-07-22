/**
 * CommendationsVsDisciplinaryBar — Side-by-side bar: commendations vs disciplinary counts.
 *
 * Visualisation: Quick moral temperature check.
 * Props: { commendationsCount, disciplinaryCount }.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

const chartConfig = {} satisfies ChartConfig;

interface Props {
    commendationsCount: number;
    disciplinaryCount: number;
    studentName?: string;
}

export function CommendationsVsDisciplinaryBar({
    commendationsCount,
    disciplinaryCount,
    studentName,
}: Props) {
    const data = [
        { label: "Commendations", value: commendationsCount, fill: "hsl(var(--chart-2))" },
        { label: "Disciplinary", value: disciplinaryCount, fill: "hsl(var(--chart-1))" },
    ];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Commendations vs Disciplinary{studentName ? ` — ${studentName}` : ""}
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={data} barCategoryGap="40%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis tickLine={false} axisLine={false} allowDecimals={false} />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
                    <Bar dataKey="value" radius={[4, 4, 0, 0]} barSize={60}>
                        {data.map((entry) => (
                            <Cell key={entry.label} fill={entry.fill} />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

export function CommendationsVsDisciplinaryBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
