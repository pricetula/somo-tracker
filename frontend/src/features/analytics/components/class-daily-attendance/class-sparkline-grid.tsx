/**
 * ClassSparklineGrid — Grid of sparkline mini-trends for all classes.
 *
 * Visualisation: Side-by-side mini attendance trend lines for all classes in a grade.
 * Props: Array of { className, data: { date, rate }[] }.
 */
"use client";

import { Line, LineChart } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const sparklineConfig = {
    rate: {
        label: "Rate",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface ClassSparklineData {
    className: string;
    latestRate: number;
    trend: "up" | "down" | "stable";
    points: { date: string; rate: number }[];
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function trendColor(trend: ClassSparklineData["trend"]) {
    if (trend === "up") return "text-emerald-600";
    if (trend === "down") return "text-destructive";
    return "text-muted-foreground";
}

function trendArrow(trend: ClassSparklineData["trend"]) {
    if (trend === "up") return "\u2191";
    if (trend === "down") return "\u2193";
    return "\u2192";
}

// ─── Component ────────────────────────────────────────────────────────────

interface ClassSparklineGridProps {
    data: ClassSparklineData[];
}

export function ClassSparklineGrid({ data }: ClassSparklineGridProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No class sparkline data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Class Attendance Trends
                <GraphHelp>
                    Mini sparkline trend charts for each class showing attendance rate over time.
                    Arrows indicate improving or declining trends.
                </GraphHelp>
            </p>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
                {data.map((cls) => (
                    <div key={cls.className} className="bg-muted/30 space-y-1 rounded p-3">
                        <div className="flex items-center justify-between">
                            <p className="text-foreground text-xs font-medium">{cls.className}</p>
                            <span
                                className={`text-xs ${trendColor(cls.trend)}`}
                                title={`${cls.trend} trend`}
                            >
                                {trendArrow(cls.trend)} {cls.latestRate.toFixed(1)}%
                            </span>
                        </div>
                        <ChartContainer
                            config={sparklineConfig}
                            className="h-8 w-full"
                            initialDimension={{ width: 120, height: 32 }}
                        >
                            <LineChart data={cls.points}>
                                <Line
                                    type="monotone"
                                    dataKey="rate"
                                    stroke="#22c55e"
                                    strokeWidth={1.5}
                                    dot={false}
                                />
                            </LineChart>
                        </ChartContainer>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
