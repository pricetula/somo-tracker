/**
 * AttendanceGauge — Gauge / Ring chart showing attendance percentage.
 *
 * Visualisation: Single student's attendance % for current term.
 * Props: Single AttendanceTermSummary record.
 */
"use client";

import { Label, PolarGrid, PolarRadiusAxis, RadialBar, RadialBarChart } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";

// ─── Helpers ──────────────────────────────────────────────────────────────

function gaugeColor(pct: number) {
    if (pct >= 90) return "hsl(var(--emerald))";
    if (pct >= 75) return "hsl(var(--amber))";
    return "hsl(var(--destructive))";
}

function gaugeLabel(pct: number) {
    if (pct >= 90) return "Excellent";
    if (pct >= 75) return "Good";
    if (pct >= 50) return "Needs Attention";
    return "Critical";
}

// ─── Config ───────────────────────────────────────────────────────────────

function buildConfig(pct: number): ChartConfig {
    return {
        attendance: {
            label: "Attendance",
            color: gaugeColor(pct),
        },
    };
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceGaugeProps {
    percentage: number;
    studentName?: string;
    learningAreaName?: string;
}

export function AttendanceGauge({
    percentage,
    studentName,
    learningAreaName,
}: AttendanceGaugeProps) {
    const chartData = [{ name: "attendance", value: percentage, fill: gaugeColor(percentage) }];
    const config = buildConfig(percentage);
    const endAngle = 90 + (percentage / 100) * 270;

    return (
        <div className="space-y-2">
            {(studentName || learningAreaName) && (
                <p className="text-foreground text-sm font-medium">
                    {studentName && `${studentName}`}
                    {learningAreaName && ` — ${learningAreaName}`}
                </p>
            )}
            <ChartContainer config={config} className="mx-auto aspect-square max-h-[220px] w-full">
                <RadialBarChart
                    data={chartData}
                    startAngle={90}
                    endAngle={endAngle}
                    innerRadius={70}
                    outerRadius={110}
                    barSize={16}
                >
                    <PolarGrid
                        gridType="circle"
                        radialLines={false}
                        stroke="hsl(var(--border))"
                        className="first:fill-muted last:fill-background"
                    />
                    <RadialBar dataKey="value" cornerRadius={8} />
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
                                                y={(viewBox.cy ?? 0) - 8}
                                                className="fill-foreground text-3xl font-bold"
                                            >
                                                {percentage.toFixed(1)}%
                                            </tspan>
                                            <tspan
                                                x={viewBox.cx}
                                                y={(viewBox.cy ?? 0) + 20}
                                                className="fill-muted-foreground text-xs"
                                            >
                                                {gaugeLabel(percentage)}
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
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function AttendanceGaugeSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[220px] w-full animate-pulse rounded-full" />
        </div>
    );
}
