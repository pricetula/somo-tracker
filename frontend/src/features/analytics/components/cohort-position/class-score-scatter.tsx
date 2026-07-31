/**
 * ClassScoreScatter — Scatter plot of all students in a class, identifying outliers.
 *
 * Visualisation: Each dot = student, with reference lines for averages.
 * Props: Array of { studentName, score } and { classAverage, gradeAverage }.
 */
"use client";

import { CartesianGrid, ReferenceLine, Scatter, ScatterChart, XAxis, YAxis, ZAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    score: {
        label: "Score",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface StudentScatterPoint {
    studentName: string;
    score: number;
    isAboveAverage?: boolean;
}

export interface ScoreReferenceLines {
    classAverage: number;
    gradeAverage: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface ClassScoreScatterProps {
    students: StudentScatterPoint[];
    referenceLines: ScoreReferenceLines;
}

export function ClassScoreScatter({ students, referenceLines }: ClassScoreScatterProps) {
    if (!students.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No class score data available.
            </p>
        );
    }

    // Assign a simple index as X for vertical spread
    const chartData = students.map((s, i) => ({
        ...s,
        x: i + 1,
    }));

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Class Score Distribution
                <GraphHelp>
                    Scatter plot of all students in a class showing individual scores. Reference
                    lines mark class and grade averages to identify outliers.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <ScatterChart margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="x" hide type="number" domain={[0, students.length + 1]} />
                    <YAxis
                        dataKey="score"
                        name="Score"
                        type="number"
                        domain={[0, 100]}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                    />
                    <ZAxis range={[80, 80]} />

                    {/* Reference lines */}
                    <ReferenceLine
                        y={referenceLines.classAverage}
                        stroke="#22c55e"
                        strokeDasharray="4 4"
                        label={{
                            value: "Class Avg",
                            position: "right",
                            fill: "#22c55e",
                            fontSize: 10,
                        }}
                    />
                    <ReferenceLine
                        y={referenceLines.gradeAverage}
                        stroke="#f59e0b"
                        strokeDasharray="4 4"
                        label={{
                            value: "Grade Avg",
                            position: "right",
                            fill: "#f59e0b",
                            fontSize: 10,
                        }}
                    />

                    <ChartTooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        content={
                            <ChartTooltipContent
                                formatter={(val, name, _item) => {
                                    const value = Number(val);
                                    const payload = (_item as unknown as Record<string, unknown>)
                                        ?.payload as StudentScatterPoint | undefined;
                                    if (name === "score" && payload)
                                        return `${payload.studentName}: ${value.toFixed(1)}%`;
                                    return isNaN(value) ? val : `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Scatter data={chartData} fill="#22c55e" />
                </ScatterChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
