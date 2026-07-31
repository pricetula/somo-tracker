"use client";

export function WeightedToggleSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="bg-muted h-8 w-48 animate-pulse rounded" />
        </div>
    );
}
