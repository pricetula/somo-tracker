"use client";

export function LevelDonutChartSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[260px] w-full animate-pulse rounded-full" />
        </div>
    );
}
