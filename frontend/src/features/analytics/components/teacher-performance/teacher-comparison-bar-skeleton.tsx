"use client";

export function TeacherComparisonBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted aspect-3/2 w-full animate-pulse rounded" />
        </div>
    );
}
