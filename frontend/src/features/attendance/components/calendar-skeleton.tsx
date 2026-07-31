"use client";

import { Skeleton } from "@/components/ui/skeleton";

export function CalendarSkeleton() {
    return (
        <div className="w-92 rounded-md p-3">
            <div className="flex items-center justify-between px-1 pt-1 pb-4">
                <Skeleton className="h-7 w-7 rounded-md" />
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-7 w-7 rounded-md" />
            </div>
            <div className="mb-2 grid grid-cols-7 gap-1">
                {Array.from({ length: 7 }).map((_, i) => (
                    <div key={i} className="flex h-8 items-center justify-center">
                        <Skeleton className="h-3 w-4" />
                    </div>
                ))}
            </div>
            <div className="grid grid-cols-7 gap-2">
                {Array.from({ length: 35 }).map((_, i) => (
                    <div key={i} className="flex h-8 items-center justify-center">
                        <Skeleton className="h-8 w-8 rounded-md" />
                    </div>
                ))}
            </div>
        </div>
    );
}
