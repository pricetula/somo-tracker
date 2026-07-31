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
import { GraphHelp } from "@/features/analytics/components/graph-help";

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
        { label: "Commendations", value: commendationsCount, fill: "#22c55e" },
        { label: "Disciplinary", value: disciplinaryCount, fill: "#ef4444" },
    ];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Commendations vs Disciplinary{studentName ? ` — ${studentName}` : ""}
                <GraphHelp>
                    Side-by-side bar chart comparing the number of commendations versus disciplinary
                    incidents for a quick moral temperature check.
                </GraphHelp>
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
