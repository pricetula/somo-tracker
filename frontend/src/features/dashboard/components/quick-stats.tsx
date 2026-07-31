"use client";

export interface StatItem {
    label: string;
    value: number;
}
interface QuickStatsProps {
    stats: StatItem[];
    isLoading?: boolean;
}

import { StatCard } from "./stat-card";
import { SkeletonRow } from "./skeleton-row";

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
