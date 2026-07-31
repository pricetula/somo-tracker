"use client";

export function StrandGaugeSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="grid grid-cols-4 gap-3">
                {Array.from({ length: 4 }).map((_, i) => (
                    <div key={i} className="bg-muted/20 animate-pulse rounded p-2">
                        <div className="bg-muted mx-auto h-3 w-16 rounded" />
                        <div className="bg-muted mx-auto mt-1 h-16 w-16 rounded-full" />
                    </div>
                ))}
            </div>
        </div>
    );
}
