"use client";

export function BehaviorCalendarHeatmapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="bg-muted h-32 w-full animate-pulse rounded" />
        </div>
    );
}
