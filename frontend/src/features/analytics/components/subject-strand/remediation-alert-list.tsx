/**
 * RemediationAlertList — Flagged sub-strands requiring remediation, sorted by urgency.
 *
 * Visualisation: List/table of sub-strands needing intervention.
 * Props: Array of sub-strands with remediation flags.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface RemediationItem {
    subStrandName: string;
    strandName: string;
    learningAreaName: string;
    masteryPercentage: number;
    level: string;
    requiresRemediation: boolean;
    totalIndicators: number;
    belowCount: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function urgencyLevel(pct: number): { label: string; className: string } {
    if (pct < 25) return { label: "Critical", className: "bg-destructive/15 text-destructive" };
    if (pct < 40) return { label: "Urgent", className: "bg-orange-500/15 text-orange-600" };
    return { label: "Needs Review", className: "bg-amber-500/15 text-amber-600" };
}

// ─── Component ────────────────────────────────────────────────────────────

interface RemediationAlertListProps {
    data: RemediationItem[];
    maxItems?: number;
}

export function RemediationAlertList({ data, maxItems = 10 }: RemediationAlertListProps) {
    const flagged = data
        .filter((item) => item.requiresRemediation)
        .sort((a, b) => a.masteryPercentage - b.masteryPercentage)
        .slice(0, maxItems);

    if (!flagged.length) {
        return (
            <div className="space-y-1">
                <p className="text-foreground text-sm font-medium">Remediation Needed</p>
                <p className="text-xs text-emerald-600">All sub-strands on track</p>
            </div>
        );
    }

    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between">
                <p className="text-foreground text-sm font-medium">Remediation Needed</p>
                <span className="bg-destructive/15 text-destructive rounded-full px-2 py-0.5 text-xs font-medium">
                    {flagged.length} {flagged.length === 1 ? "area" : "areas"}
                </span>
            </div>

            <div className="space-y-1">
                {flagged.map((item) => {
                    const urgency = urgencyLevel(item.masteryPercentage);
                    return (
                        <div
                            key={`${item.learningAreaName}-${item.subStrandName}`}
                            className="bg-muted/20 flex items-center justify-between rounded p-2"
                        >
                            <div className="min-w-0 flex-1">
                                <p className="text-foreground truncate text-xs font-medium">
                                    {item.subStrandName}
                                </p>
                                <p className="text-muted-foreground truncate text-[10px]">
                                    {item.learningAreaName} — {item.strandName}
                                </p>
                            </div>
                            <div className="flex items-center gap-2">
                                <span className="text-muted-foreground text-xs tabular-nums">
                                    {item.masteryPercentage.toFixed(0)}%
                                </span>
                                <span
                                    className={cn(
                                        "inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium",
                                        urgency.className
                                    )}
                                >
                                    {urgency.label}
                                </span>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function RemediationAlertListSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="space-y-1">
                {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="bg-muted/20 h-10 w-full animate-pulse rounded" />
                ))}
            </div>
        </div>
    );
}
