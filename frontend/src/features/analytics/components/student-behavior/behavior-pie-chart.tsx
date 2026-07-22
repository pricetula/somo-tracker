/**
 * BehaviorPieChart — Proportion of behaviour types overall (commendations vs disciplinary).
 */
"use client";

import { Pie, PieChart } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

const chartConfig = {
    commendations: { label: "Commendations", color: "hsl(var(--chart-2))" },
    disciplinary: { label: "Disciplinary", color: "hsl(var(--chart-1))" },
} satisfies ChartConfig;

interface Props {
    commendationsCount: number;
    disciplinaryCount: number;
}

export function BehaviorPieChart({ commendationsCount, disciplinaryCount }: Props) {
    const total = commendationsCount + disciplinaryCount;
    if (total === 0)
        return <p className="text-muted-foreground py-8 text-center text-sm">No behaviour data.</p>;

    const data = [
        { name: "commendations", value: commendationsCount, fill: "var(--color-commendations)" },
        { name: "disciplinary", value: disciplinaryCount, fill: "var(--color-disciplinary)" },
    ].filter((d) => d.value > 0);

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Behaviour Composition</p>
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[240px] w-full"
            >
                <PieChart>
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                formatter={(val: unknown) => {
                                    const v = Number(val);
                                    return isNaN(v) ? "" : `${v} incident${v !== 1 ? "s" : ""}`;
                                }}
                            />
                        }
                    />
                    <Pie
                        data={data}
                        dataKey="value"
                        nameKey="name"
                        innerRadius={50}
                        outerRadius={90}
                        strokeWidth={2}
                        stroke="hsl(var(--background))"
                        paddingAngle={2}
                    />
                </PieChart>
            </ChartContainer>
            <p className="text-muted-foreground text-center text-xs">{total} total incidents</p>
        </div>
    );
}

export function BehaviorPieChartSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[240px] w-full animate-pulse rounded-full" />
        </div>
    );
}
