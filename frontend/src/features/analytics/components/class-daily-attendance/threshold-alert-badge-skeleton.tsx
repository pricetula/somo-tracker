"use client";

export function ThresholdAlertBadgeSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="bg-muted h-12 w-full animate-pulse rounded" />
        </div>
    );
}
