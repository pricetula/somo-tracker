"use client";

export function TeacherKpiCardsSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-36 animate-pulse rounded" />
            <div className="grid grid-cols-5 gap-3">
                {Array.from({ length: 5 }).map((_, i) => (
                    <div key={i} className="bg-muted/20 animate-pulse rounded p-3">
                        <div className="bg-muted h-8 w-full rounded" />
                    </div>
                ))}
            </div>
        </div>
    );
}
