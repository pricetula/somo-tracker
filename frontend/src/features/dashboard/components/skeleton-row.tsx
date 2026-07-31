"use client";

import { StatCardSkeleton } from "./stat-card-skeleton";

export function SkeletonRow({ count }: { count: number }) {
    return (
        <div className="flex flex-wrap gap-x-10 gap-y-4">
            {Array.from({ length: count }).map((_, i) => (
                <StatCardSkeleton key={i} />
            ))}
        </div>
    );
}
