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
import { GraphHelp } from "@/features/analytics/components/graph-help";

const chartConfig = {
    commendations: { label: "Commendations", color: "#22c55e" },
    disciplinary: { label: "Disciplinary", color: "#ef4444" },
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
        { name: "commendations", value: commendationsCount, fill: "#22c55e" },
        { name: "disciplinary", value: disciplinaryCount, fill: "#ef4444" },
    ].filter((d) => d.value > 0);

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Behaviour Composition
                <GraphHelp>
                    Donut chart showing the proportion of commendations versus disciplinary
                    incidents for the student.
                </GraphHelp>
            </p>
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
                        stroke="#ffffff"
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
