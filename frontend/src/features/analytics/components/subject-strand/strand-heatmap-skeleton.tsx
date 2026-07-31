"use client";

export function StrandHeatmapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted h-48 w-full animate-pulse rounded" />
        </div>
    );
}
