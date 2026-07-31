"use client";

interface ThresholdAlertListProps {
    alerts: ThresholdAlert[];
}
export interface ThresholdAlert {
    date: string;
    dateLabel: string;
    rate: number;
    className?: string;
    threshold?: number;
}

import { ThresholdAlertBadge } from "./threshold-alert-badge";

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
