/**
 * TeacherKpiCards — Grid of KPI cards showing each metric with value + trend.
 */
"use client";

import { cn } from "@/lib/utils";

export interface TeacherKpiMetric {
    label: string;
    value: number;
    suffix: string;
    trend?: "up" | "down" | "stable";
    color?: string;
}

interface Props {
    data: TeacherKpiMetric[];
}

function trendArrow(trend?: string) {
    if (trend === "up") return "\u2191";
    if (trend === "down") return "\u2193";
    return "\u2192";
}

function trendColor(trend?: string) {
    if (trend === "up") return "text-emerald-600";
    if (trend === "down") return "text-destructive";
    return "text-muted-foreground";
}

export function TeacherKpiCards({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No KPI data.</p>;
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Performance KPIs</p>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
                {data.map((kpi) => (
                    <div key={kpi.label} className="bg-muted/20 space-y-1 rounded p-3">
                        <p className="text-muted-foreground truncate text-xs">{kpi.label}</p>
                        <div className="flex items-baseline gap-1">
                            <span className="text-foreground text-2xl font-bold tabular-nums">
                                {kpi.value.toFixed(kpi.suffix === "%" ? 1 : 1)}
                            </span>
                            <span className="text-muted-foreground text-xs">{kpi.suffix}</span>
                        </div>
                        {kpi.trend && (
                            <span className={cn("text-xs", trendColor(kpi.trend))}>
                                {trendArrow(kpi.trend)} {kpi.trend}
                            </span>
                        )}
                    </div>
                ))}
            </div>
        </div>
    );
}

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
