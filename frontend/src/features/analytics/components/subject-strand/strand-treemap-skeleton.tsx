"use client";

export function StrandTreemapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted aspect-[4/3] w-full animate-pulse rounded" />
        </div>
    );
}
