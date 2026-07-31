"use client";

export function RemediationAlertListSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="space-y-1">
                {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="bg-muted/20 h-10 w-full animate-pulse rounded" />
                ))}
            </div>
        </div>
    );
}
