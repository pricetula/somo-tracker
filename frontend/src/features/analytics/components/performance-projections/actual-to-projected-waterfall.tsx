/**
 * ActualToProjectedWaterfall — Shows the projected jump/decline from current term.
 *
 * Visualisation: Simple two-bar: current score vs projected score with connecting arrow.
 * Props: { lastTermScore, projectedScore, subjectName }.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis } from "recharts";

import { GraphHelp } from "@/components/GraphHelp";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    value: {
        label: "Score",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface ActualToProjectedData {
    lastTermScore: number;
    projectedScore: number;
    subjectName?: string;
    momentumScore?: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface ActualToProjectedWaterfallProps {
    data: ActualToProjectedData;
}

export function ActualToProjectedWaterfall({ data }: ActualToProjectedWaterfallProps) {
    const { lastTermScore, projectedScore, subjectName, momentumScore } = data;
    const change = projectedScore - lastTermScore;
    const isUp = change >= 0;

    const chartData = [
        { label: "Current", value: lastTermScore },
        { label: "Projected", value: projectedScore },
    ];

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                {subjectName ? `${subjectName} — ` : ""}Current vs Projected
                <GraphHelp>
                    Two-bar chart comparing the student&rsquo;s current score to their projected
                    score for the next term.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <BarChart data={chartData} barCategoryGap="40%">
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Bar dataKey="value" radius={[4, 4, 0, 0]} barSize={60}>
                        {chartData.map((entry) => (
                            <Cell
                                key={entry.label}
                                fill={
                                    entry.label === "Current"
                                        ? "#6b7280"
                                        : isUp
                                          ? "#22c55e"
                                          : "#ef4444"
                                }
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>

            <div className="flex items-center justify-center gap-2 text-sm">
                <span className="text-muted-foreground">{lastTermScore.toFixed(1)}%</span>
                <span className={isUp ? "text-emerald-600" : "text-destructive"}>
                    {isUp ? "\u2192" : "\u2192"}
                </span>
                <span className="text-foreground font-medium">{projectedScore.toFixed(1)}%</span>
                <span
                    className={
                        isUp
                            ? "text-xs font-medium text-emerald-600"
                            : "text-destructive text-xs font-medium"
                    }
                >
                    ({isUp ? "+" : ""}
                    {change.toFixed(1)}%)
                </span>
            </div>

            {momentumScore !== undefined && (
                <p className="text-muted-foreground text-center text-xs">
                    Momentum: {momentumScore > 0 ? "+" : ""}
                    {momentumScore.toFixed(2)} pts/term
                </p>
            )}
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
