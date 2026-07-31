/**
 * ReportCardForecastCard — Next-term projected level with confidence badge.
 *
 * Visualisation: A card showing projected score, level, and confidence.
 * Props: { projectedScore, confidencePercentage, riskLevel, learningAreaName }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface ForecastCardData {
    projectedScore: number;
    confidencePercentage: number;
    riskLevel: string;
    learningAreaName?: string;
    lastTermScore?: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function projectedLevel(score: number): string {
    if (score >= 80) return "EE";
    if (score >= 60) return "ME";
    if (score >= 40) return "AE";
    return "BE";
}

function levelColor(level: string): string {
    switch (level) {
        case "EE":
            return "text-emerald-600";
        case "ME":
            return "text-blue-600";
        case "AE":
            return "text-amber-600";
        case "BE":
            return "text-destructive";
        default:
            return "text-foreground";
    }
}

function riskColor(risk: string): string {
    switch (risk) {
        case "Low":
            return "bg-emerald-500/15 text-emerald-600";
        case "Medium":
            return "bg-amber-500/15 text-amber-600";
        case "High":
            return "bg-destructive/15 text-destructive";
        default:
            return "bg-muted text-muted-foreground";
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface ReportCardForecastCardProps {
    data: ForecastCardData;
}

export function ReportCardForecastCard({ data }: ReportCardForecastCardProps) {
    const level = projectedLevel(data.projectedScore);
    const change =
        data.lastTermScore !== undefined ? data.projectedScore - data.lastTermScore : null;

    return (
        <div className="bg-muted/20 space-y-3 rounded-md p-4">
            <div className="flex items-center justify-between">
                <p className="text-foreground text-sm font-medium">Next Term Forecast</p>
                {data.learningAreaName && (
                    <span className="text-muted-foreground text-xs">{data.learningAreaName}</span>
                )}
            </div>

            <div className="flex items-baseline justify-center gap-3 py-2">
                <span className={cn("text-4xl font-bold", levelColor(level))}>
                    {data.projectedScore.toFixed(0)}%
                </span>
                <span className={cn("text-lg font-semibold", levelColor(level))}>{level}</span>
            </div>

            {change !== null && (
                <p
                    className={cn(
                        "text-center text-sm",
                        change >= 0 ? "text-emerald-600" : "text-destructive"
                    )}
                >
                    {change >= 0 ? "+" : ""}
                    {change.toFixed(1)}% from current term
                </p>
            )}

            <div className="flex items-center justify-center gap-3">
                <span
                    className={cn(
                        "inline-flex items-center rounded-full px-3 py-1 text-xs font-medium",
                        riskColor(data.riskLevel)
                    )}
                >
                    {data.riskLevel} Risk
                </span>
                <span className="text-muted-foreground text-xs">
                    {data.confidencePercentage}% confidence
                </span>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
