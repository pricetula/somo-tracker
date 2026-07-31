/**
 * ConfidenceBadge — Shows confidence level with descriptive text.
 *
 * Visualisation: "High confidence (3 terms of data)" or "Low confidence (new student)".
 * Props: { confidencePercentage, termsOfData? }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface ConfidenceData {
    confidencePercentage: number;
    termsOfData?: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function confidenceLabel(pct: number): string {
    if (pct >= 80) return "High";
    if (pct >= 50) return "Medium";
    return "Low";
}

function confidenceColor(pct: number): string {
    if (pct >= 80) return "bg-emerald-500/15 text-emerald-600";
    if (pct >= 50) return "bg-amber-500/15 text-amber-600";
    return "bg-destructive/15 text-destructive";
}

function confidenceDescription(pct: number, termsOfData?: number): string {
    if (termsOfData !== undefined) {
        if (termsOfData >= 3) return `${termsOfData} terms of data — reliable projection`;
        if (termsOfData >= 2) return `${termsOfData} terms of data — moderate confidence`;
        return `${termsOfData ?? 1} term${(termsOfData ?? 1) > 1 ? "s" : ""} of data — limited confidence`;
    }
    if (pct >= 80) return "Strong historical trend";
    if (pct >= 50) return "Some data available";
    return "Limited data — projection may change";
}

// ─── Component ────────────────────────────────────────────────────────────

interface ConfidenceBadgeProps {
    data: ConfidenceData;
    variant?: "badge" | "card";
}

export function ConfidenceBadge({ data, variant = "badge" }: ConfidenceBadgeProps) {
    const label = confidenceLabel(data.confidencePercentage);
    const color = confidenceColor(data.confidencePercentage);
    const description = confidenceDescription(data.confidencePercentage, data.termsOfData);

    if (variant === "badge") {
        return (
            <span
                className={cn(
                    "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                    color
                )}
            >
                {label} Confidence
            </span>
        );
    }

    return (
        <div
            className={cn(
                "space-y-1 rounded-md p-3",
                color
                    .replace("text", "bg")
                    .replace("emerald-600", "emerald-500/10")
                    .replace("amber-600", "amber-500/10")
                    .replace("destructive", "destructive/10")
            )}
        >
            <div className="flex items-center gap-2">
                <span
                    className={cn(
                        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                        color
                    )}
                >
                    {label} Confidence
                </span>
                <span className="text-muted-foreground text-xs">{data.confidencePercentage}%</span>
            </div>
            <p className="text-muted-foreground text-xs">{description}</p>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
