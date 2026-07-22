/**
 * RiskIndicator — Red/Amber/Green badge for projection risk level.
 *
 * Visualisation: Traffic light indicator per subject.
 * Props: { riskLevel, subjectName? }.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface RiskData {
    riskLevel: string;
    confidencePercentage: number;
    subject?: string;
    targetGapPoints?: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function riskConfig(risk: string) {
    switch (risk) {
        case "High":
            return {
                dot: "bg-destructive",
                bg: "bg-destructive/10",
                text: "text-destructive",
                label: "High Risk",
            };
        case "Medium":
            return {
                dot: "bg-amber-500",
                bg: "bg-amber-500/10",
                text: "text-amber-600",
                label: "Medium Risk",
            };
        case "Low":
            return {
                dot: "bg-emerald-500",
                bg: "bg-emerald-500/15",
                text: "text-emerald-600",
                label: "Low Risk",
            };
        default:
            return {
                dot: "bg-muted",
                bg: "bg-muted/20",
                text: "text-muted-foreground",
                label: "Unknown",
            };
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface RiskIndicatorProps {
    data: RiskData;
    variant?: "badge" | "card";
}

export function RiskIndicator({ data, variant = "badge" }: RiskIndicatorProps) {
    const config = riskConfig(data.riskLevel);

    if (variant === "badge") {
        return (
            <span
                className={cn(
                    "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium",
                    config.bg,
                    config.text
                )}
            >
                <span className={cn("h-1.5 w-1.5 rounded-full", config.dot)} />
                {config.label}
            </span>
        );
    }

    return (
        <div className={cn("space-y-1 rounded-md p-3", config.bg)}>
            {data.subject && <p className="text-foreground text-xs font-medium">{data.subject}</p>}
            <div className="flex items-center gap-2">
                <span className={cn("h-2 w-2 rounded-full", config.dot)} />
                <span className={cn("text-sm font-medium", config.text)}>{config.label}</span>
            </div>
            <p className={cn("text-xs", config.text)}>
                {data.confidencePercentage}% confidence
                {data.targetGapPoints !== undefined && (
                    <>
                        {" "}
                        — gap: {data.targetGapPoints > 0 ? "+" : ""}
                        {data.targetGapPoints.toFixed(1)} pts
                    </>
                )}
            </p>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function RiskIndicatorSkeleton() {
    return (
        <div className="bg-muted/20 animate-pulse rounded-md p-3">
            <div className="bg-muted h-4 w-24 rounded" />
        </div>
    );
}
