"use client";

export function GapBarChartSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted h-6 w-full animate-pulse rounded-full" />
        </div>
    );
}
