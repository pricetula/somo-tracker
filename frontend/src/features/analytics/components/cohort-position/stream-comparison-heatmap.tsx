/**
 * StreamComparisonHeatmap — All streams in a grade, with average scores and ranges.
 *
 * Visualisation: Heatmap grid of streams × metrics (avg score, range, etc.).
 * Props: Array of { streamName, averageScore, minScore, maxScore, studentCount }.
 */
"use client";

import { cn } from "@/lib/utils";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Types ────────────────────────────────────────────────────────────────

export interface StreamSummary {
    streamName: string;
    averageScore: number;
    minScore: number;
    maxScore: number;
    studentCount: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function heatColor(score: number): string {
    if (score >= 80) return "bg-emerald-500/20 text-emerald-600";
    if (score >= 60) return "bg-blue-500/20 text-blue-600";
    if (score >= 40) return "bg-amber-500/20 text-amber-600";
    return "bg-destructive/20 text-destructive";
}

// ─── Component ────────────────────────────────────────────────────────────

interface StreamComparisonHeatmapProps {
    data: StreamSummary[];
}

export function StreamComparisonHeatmap({ data }: StreamComparisonHeatmapProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No stream comparison data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Stream Comparison
                <GraphHelp>
                    Heatmap grid comparing all streams in a grade by average score, score range, and
                    student count. Green shades indicate higher average performance.
                </GraphHelp>
            </p>
            <div className="overflow-x-auto">
                <table className="w-full text-sm">
                    <thead>
                        <tr className="text-muted-foreground border-b text-xs uppercase">
                            <th className="px-3 py-2 text-left font-medium">Stream</th>
                            <th className="px-3 py-2 text-center font-medium">Students</th>
                            <th className="px-3 py-2 text-center font-medium">Average</th>
                            <th className="px-3 py-2 text-center font-medium">Min</th>
                            <th className="px-3 py-2 text-center font-medium">Max</th>
                            <th className="px-3 py-2 text-center font-medium">Range</th>
                        </tr>
                    </thead>
                    <tbody>
                        {data.map((stream) => (
                            <tr key={stream.streamName} className="border-b last:border-0">
                                <td className="text-foreground px-3 py-2 font-medium">
                                    {stream.streamName}
                                </td>
                                <td className="text-muted-foreground px-3 py-2 text-center">
                                    {stream.studentCount}
                                </td>
                                <td
                                    className={cn(
                                        "px-3 py-2 text-center font-medium",
                                        heatColor(stream.averageScore)
                                    )}
                                >
                                    {stream.averageScore.toFixed(1)}%
                                </td>
                                <td className="text-muted-foreground px-3 py-2 text-center">
                                    {stream.minScore.toFixed(1)}%
                                </td>
                                <td className="text-muted-foreground px-3 py-2 text-center">
                                    {stream.maxScore.toFixed(1)}%
                                </td>
                                <td className="text-muted-foreground px-3 py-2 text-center">
                                    {(stream.maxScore - stream.minScore).toFixed(1)}%
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            {/* Mini bar visualization alongside */}
            <div className="space-y-1 pt-2">
                <p className="text-muted-foreground text-xs">Score distribution by stream:</p>
                {data.map((stream) => (
                    <div key={stream.streamName} className="flex items-center gap-2">
                        <span className="text-muted-foreground w-20 text-right text-xs">
                            {stream.streamName}
                        </span>
                        <div className="bg-muted relative h-2 flex-1 overflow-hidden rounded-full">
                            <div
                                className="bg-muted-foreground/30 absolute h-full rounded-full"
                                style={{
                                    left: `${stream.minScore}%`,
                                    width: `${stream.maxScore - stream.minScore}%`,
                                }}
                            />
                            <div
                                className="bg-foreground absolute top-0 h-full w-0.5 rounded-full"
                                style={{ left: `${stream.averageScore}%` }}
                            />
                        </div>
                        <span className="text-muted-foreground w-12 text-right text-xs tabular-nums">
                            {stream.averageScore.toFixed(0)}%
                        </span>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
