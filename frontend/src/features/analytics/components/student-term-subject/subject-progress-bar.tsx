/**
 * SubjectProgressBar — Simple parent-friendly "you are here" progress bar per subject.
 *
 * Visualisation: Horizontal progress bar with level threshold markers.
 * Props: { subject, score, level, thresholds }.
 */
"use client";

import { cn } from "@/lib/utils";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Types ────────────────────────────────────────────────────────────────

export interface ProgressBarData {
    subject: string;
    score: number;
    level: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function levelColor(level: string): string {
    switch (level) {
        case "EE":
            return "bg-emerald-500";
        case "ME":
            return "bg-blue-500";
        case "AE":
            return "bg-amber-500";
        case "BE":
            return "bg-destructive/70";
        default:
            return "bg-muted";
    }
}

function levelLabel(level: string): string {
    switch (level) {
        case "EE":
            return "Exceeding";
        case "ME":
            return "Meeting";
        case "AE":
            return "Approaching";
        case "BE":
            return "Below";
        default:
            return level;
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface SubjectProgressBarProps {
    data: ProgressBarData[];
}

export function SubjectProgressBar({ data }: SubjectProgressBarProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No subject progress data available.
            </p>
        );
    }

    return (
        <div className="space-y-3">
            <p className="text-foreground text-sm font-medium">
                Progress by Subject
                <GraphHelp>
                    Horizontal progress bars for each subject with threshold markers for EE, ME, and
                    AE levels. Shows current score and level at a glance.
                </GraphHelp>
            </p>
            <div className="space-y-3">
                {data.map((entry) => (
                    <div key={entry.subject} className="space-y-1">
                        <div className="flex items-center justify-between">
                            <p className="text-foreground text-xs font-medium">{entry.subject}</p>
                            <div className="flex items-center gap-2">
                                <span className="text-muted-foreground text-xs">
                                    {levelLabel(entry.level)}
                                </span>
                                <span className="text-foreground text-xs font-medium tabular-nums">
                                    {entry.score.toFixed(1)}%
                                </span>
                            </div>
                        </div>

                        {/* Progress bar track */}
                        <div className="bg-muted relative h-3 w-full overflow-hidden rounded-full">
                            {/* Threshold markers */}
                            <div
                                className="absolute top-0 h-full w-px bg-emerald-400/50"
                                style={{ left: "80%" }}
                                title="EE threshold (80%)"
                            />
                            <div
                                className="absolute top-0 h-full w-px bg-blue-400/50"
                                style={{ left: "60%" }}
                                title="ME threshold (60%)"
                            />
                            <div
                                className="absolute top-0 h-full w-px bg-amber-400/50"
                                style={{ left: "40%" }}
                                title="AE threshold (40%)"
                            />

                            {/* Filled bar */}
                            <div
                                className={cn(
                                    "h-full rounded-full transition-all duration-500",
                                    levelColor(entry.level)
                                )}
                                style={{ width: `${Math.min(entry.score, 100)}%` }}
                            />
                        </div>

                        {/* Threshold labels */}
                        <div className="flex justify-between px-0.5">
                            <span className="text-muted-foreground/60 text-[10px]">0%</span>
                            <span className="text-[10px] text-amber-400/60">AE</span>
                            <span className="text-[10px] text-blue-400/60">ME</span>
                            <span className="text-[10px] text-emerald-400/60">EE</span>
                            <span className="text-muted-foreground/60 text-[10px]">100%</span>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
