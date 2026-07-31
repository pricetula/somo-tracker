"use client";

export function TeacherPerformanceRadarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[300px] w-full animate-pulse rounded" />
        </div>
    );
}
