"use client";

export function TopBottomListSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <div key={i} className="bg-muted h-6 w-full animate-pulse rounded" />
                    ))}
                </div>
                <div className="space-y-1">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <div key={i} className="bg-muted h-6 w-full animate-pulse rounded" />
                    ))}
                </div>
            </div>
        </div>
    );
}
