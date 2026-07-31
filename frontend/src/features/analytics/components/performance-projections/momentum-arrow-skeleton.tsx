"use client";

export function MomentumArrowSkeleton() {
    return (
        <div className="flex items-center gap-2">
            <div className="bg-muted h-3 w-20 animate-pulse rounded" />
            <div className="bg-muted h-5 w-5 animate-pulse rounded" />
        </div>
    );
}
