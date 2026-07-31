"use client";

export interface StatItem {
    label: string;
    value: number;
}

export function StatCard({ label, value }: StatItem) {
    return (
        <div className="space-y-1">
            <p className="text-muted-foreground text-sm">{label}</p>
            <p className="text-3xl font-semibold tracking-tight tabular-nums">{value}</p>
        </div>
    );
}
