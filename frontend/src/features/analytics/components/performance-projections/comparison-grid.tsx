/**
 * ComparisonGrid — All subjects in one view: projected score + risk level + trend.
 *
 * Visualisation: Card grid showing each subject's projection at a glance.
 * Props: Array of projection records.
 */
"use client";

import { Line, LineChart } from "recharts";
import { cn } from "@/lib/utils";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const sparkConfig = {
    score: {
        label: "Score",
        color: "hsl(var(--chart-2))",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface ComparisonGridItem {
    learningAreaName: string;
    lastScore: number;
    projectedScore: number;
    riskLevel: string;
    confidencePercentage: number;
    momentumScore: number;
    historicalScores: { termName: string; score: number }[];
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function riskClass(risk: string) {
    switch (risk) {
        case "Low":
            return "border-emerald-500/30";
        case "Medium":
            return "border-amber-500/30";
        case "High":
            return "border-destructive/30";
        default:
            return "border-border";
    }
}

function riskDot(risk: string) {
    switch (risk) {
        case "Low":
            return "bg-emerald-500";
        case "Medium":
            return "bg-amber-500";
        case "High":
            return "bg-destructive";
        default:
            return "bg-muted";
    }
}

function arrowSymbol(momentum: number): string {
    if (momentum > 2) return "\u2191";
    if (momentum < -2) return "\u2193";
    return "\u2192";
}

function arrowColor(momentum: number): string {
    if (momentum > 2) return "text-emerald-600";
    if (momentum < -2) return "text-destructive";
    return "text-muted-foreground";
}

// ─── Component ────────────────────────────────────────────────────────────

interface ComparisonGridProps {
    data: ComparisonGridItem[];
}

export function ComparisonGrid({ data }: ComparisonGridProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No projection comparison data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">All Subjects Overview</p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {data.map((item) => (
                    <div
                        key={item.learningAreaName}
                        className={cn(
                            "bg-muted/20 space-y-2 rounded-md border p-3",
                            riskClass(item.riskLevel)
                        )}
                    >
                        <div className="flex items-center justify-between">
                            <p className="text-foreground text-xs font-medium">
                                {item.learningAreaName}
                            </p>
                            <span
                                className={cn("text-sm font-bold", arrowColor(item.momentumScore))}
                            >
                                {arrowSymbol(item.momentumScore)}
                            </span>
                        </div>

                        {/* Sparkline */}
                        <ChartContainer
                            config={sparkConfig}
                            className="h-8 w-full"
                            initialDimension={{ width: 120, height: 32 }}
                        >
                            <LineChart data={item.historicalScores}>
                                <Line
                                    type="monotone"
                                    dataKey="score"
                                    stroke="var(--color-score)"
                                    strokeWidth={1.5}
                                    dot={false}
                                />
                            </LineChart>
                        </ChartContainer>

                        {/* Scores */}
                        <div className="flex items-center justify-between text-xs">
                            <span className="text-muted-foreground">
                                {item.lastScore.toFixed(1)}%
                            </span>
                            <span className="text-foreground font-medium">
                                {item.projectedScore.toFixed(1)}%
                            </span>
                        </div>

                        {/* Risk & confidence */}
                        <div className="flex items-center justify-between">
                            <span className="flex items-center gap-1">
                                <span
                                    className={cn(
                                        "h-1.5 w-1.5 rounded-full",
                                        riskDot(item.riskLevel)
                                    )}
                                />
                                <span className="text-muted-foreground text-[10px]">
                                    {item.riskLevel}
                                </span>
                            </span>
                            <span className="text-muted-foreground text-[10px]">
                                {item.confidencePercentage}% conf.
                            </span>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function ComparisonGridSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="grid grid-cols-3 gap-3">
                {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="bg-muted/20 animate-pulse rounded-md p-3">
                        <div className="bg-muted h-3 w-20 rounded" />
                        <div className="bg-muted mt-2 h-8 w-full rounded" />
                    </div>
                ))}
            </div>
        </div>
    );
}
