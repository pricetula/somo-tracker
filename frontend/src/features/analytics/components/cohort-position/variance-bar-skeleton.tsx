"use client";

export function VarianceBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted h-12 w-full animate-pulse rounded" />
        </div>
    );
}
