/**
 * NetSentimentScore — commendations_count - disciplinary_count (positive = good).
 */
"use client";

import { cn } from "@/lib/utils";

interface Props {
    commendationsCount: number;
    disciplinaryCount: number;
}

export function NetSentimentScore({ commendationsCount, disciplinaryCount }: Props) {
    const net = commendationsCount - disciplinaryCount;
    const isPositive = net >= 0;
    const total = commendationsCount + disciplinaryCount;

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Net Sentiment</p>
            <div className="flex items-center justify-center gap-4">
                <div className="text-center">
                    <p className="text-muted-foreground text-xs">Commendations</p>
                    <p className="text-2xl font-bold text-emerald-600">{commendationsCount}</p>
                </div>
                <div className="text-muted-foreground text-2xl">&minus;</div>
                <div className="text-center">
                    <p className="text-muted-foreground text-xs">Disciplinary</p>
                    <p className="text-destructive text-2xl font-bold">{disciplinaryCount}</p>
                </div>
                <div className="text-muted-foreground text-2xl">=</div>
                <div className="text-center">
                    <p className="text-muted-foreground text-xs">Net Score</p>
                    <p
                        className={cn(
                            "text-3xl font-bold",
                            isPositive ? "text-emerald-600" : "text-destructive"
                        )}
                    >
                        {net > 0 ? "+" : ""}
                        {net}
                    </p>
                </div>
            </div>
            {total > 0 && (
                <p className="text-muted-foreground text-center text-xs">
                    {commendationsCount}/{total} ({((commendationsCount / total) * 100).toFixed(0)}
                    %) positive incidents
                </p>
            )}
        </div>
    );
}

export function NetSentimentScoreSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-32 animate-pulse rounded" />
            <div className="bg-muted h-12 w-full animate-pulse rounded" />
        </div>
    );
}
