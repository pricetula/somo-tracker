/**
 * ClassComparisonBoxPlot — Distribution of disciplinary counts across the class.
 * Each student shown as a dot; class average highlighted.
 */
"use client";

import { CartesianGrid, Scatter, ScatterChart, XAxis, YAxis, ZAxis, ReferenceLine } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

const chartConfig = {
    count: { label: "Incidents", color: "#ef4444" },
} satisfies ChartConfig;

export interface StudentIncidentCount {
    studentName: string;
    disciplinaryCount: number;
}

interface Props {
    data: StudentIncidentCount[];
    classAverage: number;
}

export function ClassComparisonBoxPlot({ data, classAverage }: Props) {
    if (!data.length)
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No class comparison data.
            </p>
        );
    const chartData = data.map((s, i) => ({ ...s, x: i + 1 }));

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Class Disciplinary Distribution
                <GraphHelp>
                    Scatter plot showing disciplinary incident counts across all students in the
                    class. The class average line helps identify outliers needing intervention.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <ScatterChart margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="x" hide type="number" domain={[0, data.length + 1]} />
                    <YAxis
                        dataKey="disciplinaryCount"
                        name="Incidents"
                        type="number"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        allowDecimals={false}
                    />
                    <ZAxis range={[80, 80]} />
                    <ReferenceLine
                        y={classAverage}
                        stroke="#6b7280"
                        strokeDasharray="4 4"
                        label={{
                            value: "Class Avg",
                            position: "right",
                            fill: "#6b7280",
                            fontSize: 10,
                        }}
                    />
                    <ChartTooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        content={<ChartTooltipContent />}
                    />
                    <Scatter data={chartData} fill="#ef4444" />
                </ScatterChart>
            </ChartContainer>
        </div>
    );
}

export function ClassComparisonBoxPlotSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
