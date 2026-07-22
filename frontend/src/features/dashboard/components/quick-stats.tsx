/**
 * QuickStats — a row of simple count cards showing key numbers at a glance.
 *
 * Accepts stats as an array so it can be reused across roles.
 * Each stat has a label and numeric value.
 */

"use client";

import { Skeleton } from "@/components/ui/skeleton";

export interface StatItem {
    label: string;
    value: number;
}

interface QuickStatsProps {
    stats: StatItem[];
    isLoading?: boolean;
}

function StatCard({ label, value }: StatItem) {
    return (
        <div className="space-y-1">
            <p className="text-muted-foreground text-sm">{label}</p>
            <p className="text-3xl font-semibold tracking-tight tabular-nums">{value}</p>
        </div>
    );
}

function StatCardSkeleton() {
    return (
        <div className="space-y-2">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-9 w-16" />
        </div>
    );
}

function SkeletonRow({ count }: { count: number }) {
    return (
        <div className="flex flex-wrap gap-x-10 gap-y-4">
            {Array.from({ length: count }).map((_, i) => (
                <StatCardSkeleton key={i} />
            ))}
        </div>
    );
}

export function QuickStats({ stats, isLoading }: QuickStatsProps) {
    if (isLoading) {
        return (
            <section>
                <h2 className="mb-3 text-lg font-medium">At a Glance</h2>
                <SkeletonRow count={stats.length || 3} />
            </section>
        );
    }

    if (stats.length === 0) return null;

    return (
        <section>
            <h2 className="mb-3 text-lg font-medium">At a Glance</h2>
            <div className="flex flex-wrap gap-x-10 gap-y-4">
                {stats.map((stat) => (
                    <StatCard key={stat.label} label={stat.label} value={stat.value} />
                ))}
            </div>
        </section>
    );
}
