/**
 * ProjectionTableSparklines — Table with per-subject mini trend charts (sparklines).
 *
 * Visualisation: Dashboard table showing projected scores with inline sparklines.
 * Props: Array of projection records.
 */
"use client";

import { Line, LineChart } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { cn } from "@/lib/utils";

// ─── Config ───────────────────────────────────────────────────────────────

const sparkConfig = {
    score: {
        label: "Score",
        color: "hsl(var(--chart-2))",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface ProjectionTableRow {
    subjectName: string;
    lastScore: number;
    projectedScore: number;
    momentumScore: number;
    confidence: number;
    riskLevel: string;
    historicalScores: { termName: string; score: number }[];
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function riskColor(risk: string) {
    switch (risk) {
        case "Low":
            return "text-emerald-600";
        case "Medium":
            return "text-amber-600";
        case "High":
            return "text-destructive";
        default:
            return "text-muted-foreground";
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface ProjectionTableSparklinesProps {
    data: ProjectionTableRow[];
}

export function ProjectionTableSparklines({ data }: ProjectionTableSparklinesProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No projection data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Subject Projections</p>
            <div className="overflow-x-auto">
                <table className="w-full text-sm">
                    <thead>
                        <tr className="text-muted-foreground border-b text-xs uppercase">
                            <th className="px-3 py-2 text-left font-medium">Subject</th>
                            <th className="px-3 py-2 text-center font-medium">Current</th>
                            <th className="px-3 py-2 text-center font-medium">Projected</th>
                            <th className="px-3 py-2 text-center font-medium">Trend</th>
                            <th className="px-3 py-2 text-center font-medium">Risk</th>
                            <th className="px-3 py-2 text-center font-medium">Conf.</th>
                        </tr>
                    </thead>
                    <tbody>
                        {data.map((row) => (
                            <tr key={row.subjectName} className="border-b last:border-0">
                                <td className="text-foreground px-3 py-2 font-medium">
                                    {row.subjectName}
                                </td>
                                <td className="px-3 py-2 text-center tabular-nums">
                                    {row.lastScore.toFixed(1)}%
                                </td>
                                <td className="px-3 py-2 text-center font-medium tabular-nums">
                                    {row.projectedScore.toFixed(1)}%
                                </td>
                                <td className="px-3 py-2">
                                    <ChartContainer
                                        config={sparkConfig}
                                        className="h-8 w-20"
                                        initialDimension={{ width: 80, height: 32 }}
                                    >
                                        <LineChart data={row.historicalScores}>
                                            <Line
                                                type="monotone"
                                                dataKey="score"
                                                stroke="var(--color-score)"
                                                strokeWidth={1.5}
                                                dot={false}
                                            />
                                        </LineChart>
                                    </ChartContainer>
                                </td>
                                <td
                                    className={cn(
                                        "px-3 py-2 text-center text-xs font-medium",
                                        riskColor(row.riskLevel)
                                    )}
                                >
                                    {row.riskLevel}
                                </td>
                                <td className="text-muted-foreground px-3 py-2 text-center tabular-nums">
                                    {row.confidence}%
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function ProjectionTableSparklinesSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="bg-muted h-32 w-full animate-pulse rounded" />
        </div>
    );
}
