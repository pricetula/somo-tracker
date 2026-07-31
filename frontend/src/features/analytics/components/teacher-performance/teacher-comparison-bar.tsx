/**
 * TeacherComparisonBar — subject_mean_score across all teachers in a grade.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

export interface TeacherScoreEntry {
    teacherName: string;
    subjectMeanScore: number;
    subjectName?: string;
}

interface Props {
    data: TeacherScoreEntry[];
}

export function TeacherComparisonBar({ data }: Props) {
    if (!data.length)
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">No comparison data.</p>
        );
    const sorted = [...data].sort((a, b) => b.subjectMeanScore - a.subjectMeanScore);
    const chartConfig: ChartConfig = {};
    sorted.forEach((e) => {
        chartConfig[e.teacherName] = { label: e.teacherName };
    });

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Teacher Comparison
                <GraphHelp>
                    Horizontal bar chart comparing subject mean scores across all teachers in a
                    grade. Bars are colour-coded by performance level.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-3/2 w-full">
                <BarChart data={sorted} layout="vertical" barCategoryGap="20%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" domain={[0, 100]} tickLine={false} axisLine={false} />
                    <YAxis
                        dataKey="teacherName"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={120}
                        tick={{ fontSize: 11 }}
                    />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(val: unknown) => {
                                    const v = Number(val);
                                    return isNaN(v) ? "" : `${v.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="subjectMeanScore" radius={[0, 4, 4, 0]} barSize={18}>
                        {sorted.map((e) => (
                            <Cell
                                key={e.teacherName}
                                fill={
                                    e.subjectMeanScore >= 70
                                        ? "red"
                                        : e.subjectMeanScore >= 50
                                          ? "blue"
                                          : "yellow"
                                }
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}
