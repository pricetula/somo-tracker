"use client";

export function ClassSparklineGridSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="grid grid-cols-4 gap-4">
                {Array.from({ length: 4 }).map((_, i) => (
                    <div key={i} className="bg-muted/30 animate-pulse rounded p-3">
                        <div className="bg-muted h-3 w-20 rounded" />
                        <div className="bg-muted mt-2 h-8 w-full rounded" />
                    </div>
                ))}
            </div>
        </div>
    );
}
