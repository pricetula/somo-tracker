/**
 * ProjectionScatterTrend — Scatter plot with regression line and projected point.
 *
 * Visualisation: Historical scores plotted with regression line and projected next term.
 * Props: { historicalScores, projectedScore, momentumScore }.
 */
"use client";

import { CartesianGrid, Dot, Line, LineChart, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    actual: {
        label: "Actual",
        color: "#22c55e",
    },
    projected: {
        label: "Projected",
        color: "#f59e0b",
    },
    trend: {
        label: "Trend",
        color: "#6b7280",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface TrendProjectionData {
    historicalScores: { termIndex: number; termName: string; score: number }[];
    projectedScore: number;
    momentumScore: number;
    lastTermScore: number;
    learningAreaName?: string;
}

// ─── Component ────────────────────────────────────────────────────────────

interface ProjectionScatterTrendProps {
    data: TrendProjectionData;
}

export function ProjectionScatterTrend({ data }: ProjectionScatterTrendProps) {
    const { historicalScores, projectedScore } = data;

    if (!historicalScores.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No projection data available.
            </p>
        );
    }

    // Build chart data with actual points + projected
    const chartData = historicalScores.map((h) => ({
        termIndex: h.termIndex,
        termName: h.termName,
        score: h.score,
        isProjected: false,
    }));

    const nextIndex = historicalScores[historicalScores.length - 1].termIndex + 1;
    chartData.push({
        termIndex: nextIndex,
        termName: "Projected",
        score: projectedScore,
        isProjected: true,
    });

    // Regression line data (just start + end)
    const last = historicalScores[historicalScores.length - 1];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Performance Projection{data.learningAreaName ? ` — ${data.learningAreaName}` : ""}
                <GraphHelp>
                    Historical scores plotted with a regression trend line and a projected point for
                    the next term.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <LineChart data={chartData} margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid vertical={false} strokeDasharray="3 3" />
                    <XAxis dataKey="termName" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis domain={[0, 100]} tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />

                    {/* Actual score line + dots */}
                    <Line
                        type="linear"
                        dataKey="score"
                        stroke="#22c55e"
                        strokeWidth={2}
                        data={chartData.filter((d) => !d.isProjected)}
                        dot={({ payload }) => <Dot key={payload.termName} r={5} fill="#22c55e" />}
                    />

                    {/* Trend line (dashed from last actual to projected) */}
                    <Line
                        type="linear"
                        dataKey="score"
                        stroke="#6b7280"
                        strokeWidth={1.5}
                        strokeDasharray="4 4"
                        data={[
                            {
                                termIndex: last.termIndex,
                                termName: last.termName,
                                score: last.score,
                            },
                            { termIndex: nextIndex, termName: "Projected", score: projectedScore },
                        ]}
                        dot={false}
                    />
                </LineChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function ProjectionScatterTrendSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
