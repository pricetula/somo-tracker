/**
 * StrandGauge — Individual mastery gauges for each sub-strand.
 *
 * Visualisation: Compact gauge showing mastery % for one sub-strand.
 * Props: Single sub-strand summary record.
 */
"use client";

import { Label, PolarGrid, PolarRadiusAxis, RadialBar, RadialBarChart } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Types ────────────────────────────────────────────────────────────────

export interface StrandGaugeDatum {
    subStrandName: string;
    strandName: string;
    masteryPercentage: number;
    level: string;
    requiresRemediation: boolean;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function gaugeColor(pct: number): string {
    if (pct >= 80) return "#22c55e";
    if (pct >= 60) return "#3b82f6";
    if (pct >= 40) return "#f59e0b";
    return "#ef4444";
}

// ─── Component ────────────────────────────────────────────────────────────

interface StrandGaugeProps {
    data: StrandGaugeDatum[];
}

export function StrandGauge({ data }: StrandGaugeProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No strand gauge data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Sub-Strand Mastery Gauges
                <GraphHelp>
                    Compact gauges showing mastery percentage for each sub-strand. Colour
                    thresholds: green (≥80%), yellow (≥60%), orange (≥40%), red (&lt;40%).
                </GraphHelp>
            </p>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                {data.map((item) => {
                    const color = gaugeColor(item.masteryPercentage);
                    const endAngle = 90 + (item.masteryPercentage / 100) * 270;
                    const chartData = [{ value: item.masteryPercentage }];
                    const config: ChartConfig = {
                        mastery: { label: "Mastery", color },
                    };

                    return (
                        <div key={item.subStrandName} className="bg-muted/20 space-y-1 rounded p-2">
                            <p className="text-muted-foreground truncate text-center text-[10px]">
                                {item.subStrandName}
                            </p>
                            <ChartContainer
                                config={config}
                                className="mx-auto aspect-square max-h-[90px] w-full"
                                initialDimension={{ width: 90, height: 90 }}
                            >
                                <RadialBarChart
                                    data={chartData}
                                    startAngle={90}
                                    endAngle={endAngle}
                                    innerRadius={28}
                                    outerRadius={42}
                                    barSize={8}
                                >
                                    <PolarGrid
                                        gridType="circle"
                                        radialLines={false}
                                        stroke="#e5e7eb"
                                        className="first:fill-muted last:fill-background"
                                    />
                                    <RadialBar dataKey="value" cornerRadius={4} />
                                    <PolarRadiusAxis tick={false} tickLine={false} axisLine={false}>
                                        <Label
                                            content={({ viewBox }) => {
                                                if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                                                    return (
                                                        <text
                                                            x={viewBox.cx}
                                                            y={viewBox.cy}
                                                            textAnchor="middle"
                                                            dominantBaseline="middle"
                                                        >
                                                            <tspan
                                                                x={viewBox.cx}
                                                                y={viewBox.cy ?? 0}
                                                                className="fill-foreground text-xs font-bold"
                                                            >
                                                                {item.masteryPercentage.toFixed(0)}%
                                                            </tspan>
                                                        </text>
                                                    );
                                                }
                                                return null;
                                            }}
                                        />
                                    </PolarRadiusAxis>
                                </RadialBarChart>
                            </ChartContainer>
                            <p className="text-center text-[10px]">{item.level}</p>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

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
