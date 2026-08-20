"use client";

import { Skeleton } from "@/components/ui/skeleton";

export function ClassDetailSkeleton() {
    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <Skeleton className="h-8 w-8 rounded-full" />
                <div className="space-y-1.5">
                    <Skeleton className="h-6 w-48" />
                    <Skeleton className="h-4 w-32" />
                </div>
            </div>
            <div className="flex gap-3">
                <Skeleton className="h-9 w-56" />
                <Skeleton className="h-9 w-56" />
            </div>
            <div className="rounded-md border">
                <Skeleton className="h-10 w-full rounded-none border-b" />
                <Skeleton className="h-10 w-full rounded-none" />
                <Skeleton className="h-10 w-full rounded-none" />
                <Skeleton className="h-10 w-3/4 rounded-none" />
            </div>
        </div>
    );
}
