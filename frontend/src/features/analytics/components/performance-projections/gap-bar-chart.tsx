/**
 * GapBarChart — Visual distance from ME threshold — positive = green, negative = red.
 *
 * Visualisation: Horizontal bar showing gap to threshold.
 * Props: { targetGapPoints, meThreshold }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Component ────────────────────────────────────────────────────────────

interface GapBarChartProps {
    targetGapPoints: number;
    meThreshold?: number;
    subjectName?: string;
}

export function GapBarChart({ targetGapPoints, meThreshold = 60, subjectName }: GapBarChartProps) {
    const isPositive = targetGapPoints >= 0;
    const absGap = Math.abs(targetGapPoints);
    const maxGap = 50;
    const barWidth = Math.min((absGap / maxGap) * 100, 100);

    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between">
                <p className="text-foreground text-sm font-medium">
                    {subjectName ? `${subjectName} — ` : ""}Distance to ME Threshold
                </p>
                <span className="text-muted-foreground text-xs">ME = {meThreshold}%</span>
            </div>

            <div className="bg-muted relative h-6 w-full overflow-hidden rounded-full">
                {/* Center line (ME threshold) */}
                <div className="bg-foreground/30 absolute top-0 left-1/2 h-full w-0.5 -translate-x-1/2" />

                {/* Gap bar */}
                <div
                    className={cn(
                        "absolute top-0 h-full rounded-full transition-all",
                        isPositive ? "right-1/2 bg-emerald-500/60" : "bg-destructive/60 left-1/2"
                    )}
                    style={{
                        [isPositive ? "right" : "left"]: "50%",
                        width: `${Math.min(barWidth, 50)}%`,
                    }}
                />
            </div>

            <div className="flex justify-between text-xs">
                <span className="text-muted-foreground">Below</span>
                <span
                    className={cn(
                        "font-medium",
                        isPositive ? "text-emerald-600" : "text-destructive"
                    )}
                >
                    {isPositive ? "+" : ""}
                    {targetGapPoints.toFixed(1)} pts
                </span>
                <span className="text-muted-foreground">Above</span>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function GapBarChartSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted h-6 w-full animate-pulse rounded-full" />
        </div>
    );
}
