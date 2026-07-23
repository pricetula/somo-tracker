/**
 * VarianceBar — Green/red bar showing distance from grade mean.
 *
 * Visualisation: Positive (above average) = green, negative = red.
 * Props: { variance } — positive number means above average.
 */
"use client";

import { cn } from "@/lib/utils";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Component ────────────────────────────────────────────────────────────

interface VarianceBarProps {
    variance: number;
    studentScore?: number;
    gradeAverage?: number;
}

export function VarianceBar({ variance, studentScore, gradeAverage }: VarianceBarProps) {
    const isPositive = variance >= 0;
    const absVariance = Math.abs(variance);
    const barWidth = Math.min(Math.abs(variance) / 50, 1) * 100; // scale: 50pts = full bar

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Variance from Grade Average
                <GraphHelp>
                    Green/red bar showing the distance between the student&rsquo;s score and the
                    grade average. Green = above average, red = below average.
                </GraphHelp>
            </p>

            {/* Score comparison */}
            <div className="flex items-center justify-center gap-4 py-2">
                {studentScore !== undefined && (
                    <div className="text-center">
                        <p className="text-muted-foreground text-xs">Your Score</p>
                        <p
                            className={cn(
                                "text-2xl font-bold",
                                isPositive ? "text-emerald-600" : "text-destructive"
                            )}
                        >
                            {studentScore.toFixed(1)}%
                        </p>
                    </div>
                )}
                <div className="text-muted-foreground text-sm">&minus;</div>
                {gradeAverage !== undefined && (
                    <div className="text-center">
                        <p className="text-muted-foreground text-xs">Grade Avg</p>
                        <p className="text-foreground text-2xl font-bold">
                            {gradeAverage.toFixed(1)}%
                        </p>
                    </div>
                )}
                <div className="text-muted-foreground text-sm">=</div>
                <div className="text-center">
                    <p className="text-muted-foreground text-xs">Variance</p>
                    <p
                        className={cn(
                            "text-2xl font-bold",
                            isPositive ? "text-emerald-600" : "text-destructive"
                        )}
                    >
                        {isPositive ? "+" : ""}
                        {variance.toFixed(1)}%
                    </p>
                </div>
            </div>

            {/* Visual bar */}
            <div className="bg-muted relative h-4 w-full overflow-hidden rounded-full">
                {/* Center line */}
                <div className="bg-foreground/30 absolute top-0 left-1/2 h-full w-0.5 -translate-x-1/2" />

                <div
                    className={cn(
                        "absolute top-0 h-full rounded-full transition-all",
                        isPositive ? "right-1/2 bg-emerald-500/60" : "bg-destructive/60 left-1/2"
                    )}
                    style={{
                        [isPositive ? "right" : "left"]: "50%",
                        [isPositive ? "width" : "width"]: `${Math.min(barWidth, 50)}%`,
                    }}
                />
            </div>

            <p
                className={cn(
                    "text-center text-xs font-medium",
                    isPositive ? "text-emerald-600" : "text-destructive"
                )}
            >
                {isPositive
                    ? `${absVariance.toFixed(1)} points above grade average`
                    : `${absVariance.toFixed(1)} points below grade average`}
            </p>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function VarianceBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted h-12 w-full animate-pulse rounded" />
        </div>
    );
}
