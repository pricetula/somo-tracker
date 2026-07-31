"use client";

export function ComparisonGridSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="grid grid-cols-3 gap-3">
                {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="bg-muted/20 animate-pulse rounded-md p-3">
                        <div className="bg-muted h-3 w-20 rounded" />
                        <div className="bg-muted mt-2 h-8 w-full rounded" />
                    </div>
                ))}
            </div>
        </div>
    );
}
