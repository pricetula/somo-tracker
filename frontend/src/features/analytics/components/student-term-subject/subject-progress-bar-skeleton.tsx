"use client";

export function SubjectProgressBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="space-y-3">
                {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="space-y-1">
                        <div className="bg-muted h-3 w-24 animate-pulse rounded" />
                        <div className="bg-muted h-3 w-full animate-pulse rounded-full" />
                    </div>
                ))}
            </div>
        </div>
    );
}
