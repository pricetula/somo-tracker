/**
 * StrandHeatmap — Colour-coded grid of all sub-strands, one row per strand.
 *
 * Visualisation: Sub-strand × performance level heatmap.
 * Props: Grouped data with strand/sub-strand/level breakdown.
 */
"use client";

import { cn } from "@/lib/utils";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Types ────────────────────────────────────────────────────────────────

export interface StrandHeatmapCell {
    subStrandName: string;
    strandName: string;
    masteryPercentage: number;
    mappedPerformanceLevel: string;
    requiresRemediation: boolean;
}

export interface StrandHeatmapData {
    learningAreaName: string;
    cells: StrandHeatmapCell[];
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function masteryColor(pct: number): string {
    if (pct >= 80) return "bg-emerald-500/80 text-emerald-950";
    if (pct >= 60) return "bg-blue-500/60 text-blue-950";
    if (pct >= 40) return "bg-amber-500/50 text-amber-950";
    return "bg-destructive/50 text-destructive-foreground";
}

// ─── Component ────────────────────────────────────────────────────────────

interface StrandHeatmapProps {
    data: StrandHeatmapData;
}

export function StrandHeatmap({ data }: StrandHeatmapProps) {
    const { learningAreaName, cells } = data;

    if (!cells.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No strand data available.
            </p>
        );
    }

    // Group by strand
    const grouped = new Map<string, StrandHeatmapCell[]>();
    for (const cell of cells) {
        const existing = grouped.get(cell.strandName) ?? [];
        existing.push(cell);
        grouped.set(cell.strandName, existing);
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Strand Mastery Heatmap — {learningAreaName}
                <GraphHelp>
                    Colour-coded grid of all sub-strands, one row per strand. Green = high mastery,
                    red = needs remediation.
                </GraphHelp>
            </p>
            <div className="overflow-x-auto">
                <table className="w-full text-sm">
                    <thead>
                        <tr className="text-muted-foreground border-b text-xs uppercase">
                            <th className="px-3 py-2 text-left font-medium">Strand</th>
                            <th className="px-3 py-2 text-left font-medium">Sub-Strand</th>
                            <th className="px-3 py-2 text-center font-medium">Mastery</th>
                            <th className="px-3 py-2 text-center font-medium">Level</th>
                            <th className="px-3 py-2 text-center font-medium">Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        {Array.from(grouped.entries()).map(([strandName, strandCells]) =>
                            strandCells.map((cell, idx) => (
                                <tr
                                    key={`${strandName}-${cell.subStrandName}`}
                                    className="border-b last:border-0"
                                >
                                    {idx === 0 && (
                                        <td
                                            className="text-foreground px-3 py-2 text-xs font-medium"
                                            rowSpan={strandCells.length}
                                        >
                                            {strandName}
                                        </td>
                                    )}
                                    <td className="text-muted-foreground px-3 py-2">
                                        {cell.subStrandName}
                                    </td>
                                    <td className="px-3 py-2">
                                        <div className="bg-muted relative h-5 w-full overflow-hidden rounded-full">
                                            <div
                                                className={cn(
                                                    "h-full rounded-full",
                                                    masteryColor(cell.masteryPercentage)
                                                )}
                                                style={{ width: `${cell.masteryPercentage}%` }}
                                            />
                                        </div>
                                    </td>
                                    <td
                                        className={cn(
                                            "px-3 py-2 text-center text-xs font-medium",
                                            cell.mappedPerformanceLevel === "BE" &&
                                                "text-destructive",
                                            cell.mappedPerformanceLevel === "AE" &&
                                                "text-amber-600",
                                            cell.mappedPerformanceLevel === "ME" && "text-blue-600",
                                            cell.mappedPerformanceLevel === "EE" &&
                                                "text-emerald-600"
                                        )}
                                    >
                                        {cell.mappedPerformanceLevel}
                                    </td>
                                    <td className="px-3 py-2 text-center">
                                        {cell.requiresRemediation ? (
                                            <span className="bg-destructive/15 text-destructive inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium">
                                                Remediate
                                            </span>
                                        ) : (
                                            <span className="text-[10px] text-emerald-600">
                                                On Track
                                            </span>
                                        )}
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Legend */}
            <div className="flex items-center gap-4 pt-1">
                <span className="text-muted-foreground text-xs">Mastery:</span>
                {[
                    { label: "≥80%", cls: "bg-emerald-500/80" },
                    { label: "≥60%", cls: "bg-blue-500/60" },
                    { label: "≥40%", cls: "bg-amber-500/50" },
                    { label: "<40%", cls: "bg-destructive/50" },
                ].map((entry) => (
                    <div key={entry.label} className="flex items-center gap-1">
                        <div className={cn("h-3 w-6 rounded", entry.cls)} />
                        <span className="text-muted-foreground text-[10px]">{entry.label}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function StrandHeatmapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted h-48 w-full animate-pulse rounded" />
        </div>
    );
}
