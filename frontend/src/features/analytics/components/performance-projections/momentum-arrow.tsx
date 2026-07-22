/**
 * MomentumArrow — Up/Flat/Down arrow per subject showing trend direction.
 *
 * Visualisation: Simple directional indicator with value.
 * Props: { momentumScore, subjectName }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface MomentumData {
    momentumScore: number;
    subjectName?: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function momentumConfig(score: number) {
    if (score > 2) return { arrow: "\u2191", label: "Improving", cls: "text-emerald-600" };
    if (score < -2) return { arrow: "\u2193", label: "Declining", cls: "text-destructive" };
    return { arrow: "\u2192", label: "Stable", cls: "text-muted-foreground" };
}

// ─── Component ────────────────────────────────────────────────────────────

interface MomentumArrowProps {
    data: MomentumData;
}

export function MomentumArrow({ data }: MomentumArrowProps) {
    const config = momentumConfig(data.momentumScore);

    return (
        <div className="flex items-center gap-2">
            {data.subjectName && (
                <span className="text-muted-foreground text-xs">{data.subjectName}</span>
            )}
            <span className={cn("text-lg font-bold", config.cls)}>{config.arrow}</span>
            <span className={cn("text-xs font-medium", config.cls)}>{config.label}</span>
            <span className={cn("text-xs tabular-nums", config.cls)}>
                ({data.momentumScore > 0 ? "+" : ""}
                {data.momentumScore.toFixed(2)})
            </span>
        </div>
    );
}

// ─── Multiple subjects ────────────────────────────────────────────────────

interface MomentumArrowListProps {
    data: MomentumData[];
}

export function MomentumArrowList({ data }: MomentumArrowListProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-4 text-center text-sm">
                No momentum data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Subject Momentum</p>
            <div className="space-y-1">
                {data.map((item) => (
                    <MomentumArrow key={item.subjectName} data={item} />
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function MomentumArrowSkeleton() {
    return (
        <div className="flex items-center gap-2">
            <div className="bg-muted h-3 w-20 animate-pulse rounded" />
            <div className="bg-muted h-5 w-5 animate-pulse rounded" />
        </div>
    );
}
