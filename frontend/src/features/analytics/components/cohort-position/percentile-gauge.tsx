/**
 * PercentileGauge — "You are in the top X% of your grade" — parent-friendly display.
 *
 * Visualisation: Large percentage with explanatory text.
 * Props: { classPercentile, gradePercentile }.
 */
"use client";

import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Helpers ──────────────────────────────────────────────────────────────

function performanceLabel(percentile: number): string {
    if (percentile >= 95) return "Outstanding";
    if (percentile >= 80) return "Excellent";
    if (percentile >= 60) return "Good";
    if (percentile >= 40) return "Average";
    return "Needs Improvement";
}

function performanceColor(percentile: number): string {
    if (percentile >= 80) return "text-emerald-600";
    if (percentile >= 60) return "text-blue-600";
    if (percentile >= 40) return "text-amber-600";
    return "text-destructive";
}

// ─── Component ────────────────────────────────────────────────────────────

interface PercentileGaugeProps {
    classPercentile: number;
    gradePercentile: number;
    studentName?: string;
}

export function PercentileGauge({
    classPercentile,
    gradePercentile,
    studentName,
}: PercentileGaugeProps) {
    return (
        <div className="space-y-4">
            {studentName && <p className="text-foreground text-sm font-medium">{studentName}</p>}

            <p className="text-foreground text-sm font-medium">
                Percentile Rank
                <GraphHelp>
                    Shows the student&rsquo;s percentile rank within their class and grade. Higher
                    percentage means they are outperforming more peers.
                </GraphHelp>
            </p>

            <div className="grid grid-cols-2 gap-4">
                {/* Class percentile */}
                <div className="bg-muted/30 space-y-2 rounded-md p-4 text-center">
                    <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                        In Class
                    </p>
                    <p className={`text-4xl font-bold ${performanceColor(classPercentile)}`}>
                        {classPercentile.toFixed(0)}%
                    </p>
                    <p className={`text-xs font-medium ${performanceColor(classPercentile)}`}>
                        {performanceLabel(classPercentile)}
                    </p>
                    <p className="text-muted-foreground text-xs">
                        Top {classPercentile.toFixed(0)}% of your class
                    </p>
                </div>

                {/* Grade percentile */}
                <div className="bg-muted/30 space-y-2 rounded-md p-4 text-center">
                    <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                        In Grade
                    </p>
                    <p className={`text-4xl font-bold ${performanceColor(gradePercentile)}`}>
                        {gradePercentile.toFixed(0)}%
                    </p>
                    <p className={`text-xs font-medium ${performanceColor(gradePercentile)}`}>
                        {performanceLabel(gradePercentile)}
                    </p>
                    <p className="text-muted-foreground text-xs">
                        Top {gradePercentile.toFixed(0)}% of your grade
                    </p>
                </div>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function PercentileGaugeSkeleton() {
    return (
        <div className="space-y-4">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="grid grid-cols-2 gap-4">
                <div className="bg-muted/30 animate-pulse rounded-md p-4">
                    <div className="bg-muted mx-auto h-3 w-16 rounded" />
                    <div className="bg-muted mx-auto mt-2 h-10 w-20 rounded" />
                </div>
                <div className="bg-muted/30 animate-pulse rounded-md p-4">
                    <div className="bg-muted mx-auto h-3 w-16 rounded" />
                    <div className="bg-muted mx-auto mt-2 h-10 w-20 rounded" />
                </div>
            </div>
        </div>
    );
}
