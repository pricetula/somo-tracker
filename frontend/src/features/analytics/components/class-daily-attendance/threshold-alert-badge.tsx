/**
 * ThresholdAlertBadge — Red badge when a class drops below 80% attendance.
 *
 * Visualisation: Inline alert indicators for low-attendance days.
 * Props: Current attendance rate and optional metadata.
 */
"use client";

import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface ThresholdAlert {
    date: string;
    dateLabel: string;
    rate: number;
    className?: string;
    threshold?: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface ThresholdAlertBadgeProps {
    currentRate: number;
    threshold?: number;
    className?: string;
    classLabel?: string;
    dateLabel?: string;
    /** Show as a full alert card instead of a small badge */
    variant?: "badge" | "card";
}

export function ThresholdAlertBadge({
    currentRate,
    threshold = 80,
    className: externalClassName,
    classLabel,
    dateLabel,
    variant = "badge",
}: ThresholdAlertBadgeProps) {
    const isBelow = currentRate < threshold;
    const isCritical = currentRate < 60;

    if (!isBelow) return null;

    const badgeClass = cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
        isCritical ? "bg-destructive/15 text-destructive" : "bg-amber-500/15 text-amber-600",
        externalClassName
    );

    const label = isCritical
        ? `Critical: ${currentRate.toFixed(1)}%`
        : `Low: ${currentRate.toFixed(1)}%`;

    if (variant === "badge") {
        return (
            <span className={badgeClass} title={`Attendance below ${threshold}%`}>
                <span className="h-1.5 w-1.5 rounded-full bg-current" />
                {label}
            </span>
        );
    }

    return (
        <div
            className={cn(
                "space-y-1 rounded-md p-3",
                isCritical
                    ? "bg-destructive/10 text-destructive"
                    : "bg-amber-500/10 text-amber-600",
                externalClassName
            )}
        >
            <div className="flex items-center gap-2">
                <span className="h-2 w-2 rounded-full bg-current" />
                <p className="text-sm font-medium">Attendance Alert</p>
            </div>
            <p className="text-sm">
                {classLabel && `${classLabel} — `}
                {dateLabel && `${dateLabel}: `}
                {currentRate.toFixed(1)}% attendance
                {threshold && ` (below ${threshold}% threshold)`}
            </p>
        </div>
    );
}

// ─── Alert List ───────────────────────────────────────────────────────────

interface ThresholdAlertListProps {
    alerts: ThresholdAlert[];
}

export function ThresholdAlertList({ alerts }: ThresholdAlertListProps) {
    const active = alerts.filter((a) => a.rate < (a.threshold ?? 80));

    if (!active.length) {
        return (
            <div className="space-y-1">
                <p className="text-foreground text-sm font-medium">Attendance Alerts</p>
                <p className="text-xs text-emerald-600">All classes above threshold</p>
            </div>
        );
    }

    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between">
                <p className="text-foreground text-sm font-medium">Attendance Alerts</p>
                <span className="bg-destructive/15 text-destructive rounded-full px-2 py-0.5 text-xs font-medium">
                    {active.length} {active.length === 1 ? "alert" : "alerts"}
                </span>
            </div>
            <div className="space-y-1">
                {active.map((alert) => (
                    <ThresholdAlertBadge
                        key={`${alert.date}-${alert.className}`}
                        currentRate={alert.rate}
                        threshold={alert.threshold}
                        classLabel={alert.className}
                        dateLabel={alert.dateLabel}
                        variant="card"
                    />
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function ThresholdAlertBadgeSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="bg-muted h-12 w-full animate-pulse rounded" />
        </div>
    );
}
