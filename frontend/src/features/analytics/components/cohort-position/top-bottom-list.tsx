/**
 * TopBottomList — Class top N and bottom N students with scores.
 *
 * Visualisation: Side-by-side lists of top and bottom performers.
 * Props: { top: [{ rank, name, score }], bottom: [{ rank, name, score }] }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface RankedStudent {
    rank: number;
    name: string;
    score: number;
}

export interface TopBottomData {
    top: RankedStudent[];
    bottom: RankedStudent[];
    classAverage?: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function scoreColor(score: number, classAverage?: number): string {
    if (classAverage !== undefined) {
        if (score >= classAverage + 10) return "text-emerald-600";
        if (score >= classAverage) return "text-emerald-500";
        if (score >= classAverage - 10) return "text-amber-600";
        return "text-destructive";
    }
    if (score >= 80) return "text-emerald-600";
    if (score >= 60) return "text-foreground";
    return "text-destructive";
}

// ─── Component ────────────────────────────────────────────────────────────

interface TopBottomListProps {
    data: TopBottomData;
    showCount?: number;
}

export function TopBottomList({ data, showCount = 5 }: TopBottomListProps) {
    const topItems = data.top.slice(0, showCount);
    const bottomItems = data.bottom.slice(0, showCount);

    if (!topItems.length && !bottomItems.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No ranking data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Class Rankings</p>

            <div className="grid grid-cols-2 gap-4">
                {/* Top performers */}
                <div className="space-y-1">
                    <p className="text-xs font-medium tracking-wide text-emerald-600 uppercase">
                        Top {showCount}
                    </p>
                    <div className="space-y-1">
                        {topItems.map((student) => (
                            <div
                                key={student.rank}
                                className="flex items-center justify-between rounded px-2 py-1"
                            >
                                <div className="flex items-center gap-2">
                                    <span className="text-muted-foreground w-5 text-right text-xs tabular-nums">
                                        #{student.rank}
                                    </span>
                                    <span className="text-foreground text-sm">{student.name}</span>
                                </div>
                                <span
                                    className={cn(
                                        "text-xs font-medium tabular-nums",
                                        scoreColor(student.score, data.classAverage)
                                    )}
                                >
                                    {student.score.toFixed(1)}%
                                </span>
                            </div>
                        ))}
                    </div>
                </div>

                {/* Bottom performers */}
                <div className="space-y-1">
                    <p className="text-destructive text-xs font-medium tracking-wide uppercase">
                        Bottom {showCount}
                    </p>
                    <div className="space-y-1">
                        {bottomItems.map((student) => (
                            <div
                                key={student.rank}
                                className="flex items-center justify-between rounded px-2 py-1"
                            >
                                <div className="flex items-center gap-2">
                                    <span className="text-muted-foreground w-5 text-right text-xs tabular-nums">
                                        #{student.rank}
                                    </span>
                                    <span className="text-foreground text-sm">{student.name}</span>
                                </div>
                                <span
                                    className={cn(
                                        "text-xs font-medium tabular-nums",
                                        scoreColor(student.score, data.classAverage)
                                    )}
                                >
                                    {student.score.toFixed(1)}%
                                </span>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function TopBottomListSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <div key={i} className="bg-muted h-6 w-full animate-pulse rounded" />
                    ))}
                </div>
                <div className="space-y-1">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <div key={i} className="bg-muted h-6 w-full animate-pulse rounded" />
                    ))}
                </div>
            </div>
        </div>
    );
}
