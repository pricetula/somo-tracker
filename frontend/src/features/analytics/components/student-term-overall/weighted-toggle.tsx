/**
 * WeightedToggle — Shows or hides KNEC formula impact on the score.
 *
 * Visualisation: Toggle between weighted and unweighted score display.
 * Props: { unweightedScore, weightedScore, isWeighted, onToggle }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Component ────────────────────────────────────────────────────────────

interface WeightedToggleProps {
    unweightedScore: number;
    weightedScore?: number | null;
    isWeighted: boolean;
    onToggle: (useWeighted: boolean) => void;
}

export function WeightedToggle({
    unweightedScore,
    weightedScore,
    isWeighted,
    onToggle,
}: WeightedToggleProps) {
    const hasWeightedData = weightedScore !== null && weightedScore !== undefined;

    if (!hasWeightedData) {
        return (
            <div className="space-y-1">
                <p className="text-foreground text-sm font-medium">Overall Score</p>
                <p className="text-foreground text-2xl font-bold">{unweightedScore.toFixed(1)}%</p>
                <p className="text-muted-foreground text-xs">Unweighted (regular term)</p>
            </div>
        );
    }

    const displayScore = isWeighted ? weightedScore! : unweightedScore;
    const difference = weightedScore! - unweightedScore;

    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between">
                <p className="text-foreground text-sm font-medium">Overall Score</p>
                <div className="bg-muted inline-flex items-center rounded-full p-0.5">
                    <button
                        type="button"
                        onClick={() => onToggle(false)}
                        className={cn(
                            "rounded-full px-3 py-1 text-xs transition-colors",
                            !isWeighted
                                ? "bg-background text-foreground shadow-sm"
                                : "text-muted-foreground hover:text-foreground"
                        )}
                    >
                        Unweighted
                    </button>
                    <button
                        type="button"
                        onClick={() => onToggle(true)}
                        className={cn(
                            "rounded-full px-3 py-1 text-xs transition-colors",
                            isWeighted
                                ? "bg-background text-foreground shadow-sm"
                                : "text-muted-foreground hover:text-foreground"
                        )}
                    >
                        Weighted
                    </button>
                </div>
            </div>

            <div className="flex items-baseline gap-2">
                <p className="text-foreground text-3xl font-bold">{displayScore.toFixed(1)}%</p>
                {difference !== 0 && (
                    <span
                        className={cn(
                            "text-xs font-medium",
                            difference > 0 ? "text-emerald-600" : "text-destructive"
                        )}
                    >
                        {difference > 0 ? "+" : ""}
                        {difference.toFixed(1)}% vs unweighted
                    </span>
                )}
            </div>
            <p className="text-muted-foreground text-xs">
                {isWeighted ? "Weighted (KNEC exam formula)" : "Unweighted (regular term average)"}
            </p>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
