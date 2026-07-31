/**
 * OverallGauge — Gauge chart showing overall mean % with EE/ME/AE/BE level bands.
 *
 * Visualisation: Single overall percentage with colour-coded level zones.
 * Props: { overallMeanPercentage, mappedPerformanceLevel, studentName }.
 */
"use client";

import { Label, PolarGrid, PolarRadiusAxis, RadialBar, RadialBarChart } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Level configuration ───────────────────────────────────────────────────

const LEVEL_BANDS = [
    { min: 80, max: 100, label: "EE", color: "#22c55e" },
    { min: 60, max: 80, label: "ME", color: "#3b82f6" },
    { min: 40, max: 60, label: "AE", color: "#f59e0b" },
    { min: 0, max: 40, label: "BE", color: "#ef4444" },
] as const;

function getLevelBand(score: number) {
    for (const band of LEVEL_BANDS) {
        if (score >= band.min && score < band.max) return band;
    }
    return LEVEL_BANDS[3]; // default BE
}

function levelDescription(level: string): string {
    switch (level) {
        case "EE":
            return "Exceeding Expectation";
        case "ME":
            return "Meeting Expectation";
        case "AE":
            return "Approaching Expectation";
        case "BE":
            return "Below Expectation";
        default:
            return level;
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface OverallGaugeProps {
    overallMeanPercentage: number;
    mappedPerformanceLevel: string;
    studentName?: string;
}

export function OverallGauge({
    overallMeanPercentage,
    mappedPerformanceLevel,
    studentName,
}: OverallGaugeProps) {
    const band = getLevelBand(overallMeanPercentage);
    const endAngle = 90 + (overallMeanPercentage / 100) * 270;

    const chartConfig: ChartConfig = {
        score: {
            label: "Overall Score",
            color: band.color,
        },
    };

    const chartData = [{ name: "score", value: overallMeanPercentage }];

    return (
        <div className="space-y-2">
            {studentName && <p className="text-foreground text-sm font-medium">{studentName}</p>}
            {!studentName && (
                <p className="text-foreground text-sm font-medium">
                    Overall Performance
                    <GraphHelp>
                        Gauge chart showing the overall mean percentage with colour-coded level
                        bands: EE (green), ME (blue), AE (amber), BE (red).
                    </GraphHelp>
                </p>
            )}
            <ChartContainer
                config={chartConfig}
                className="mx-auto aspect-square max-h-[260px] w-full"
            >
                <RadialBarChart
                    data={chartData}
                    startAngle={90}
                    endAngle={endAngle}
                    innerRadius={80}
                    outerRadius={130}
                    barSize={20}
                >
                    {/* Background bands */}
                    <PolarGrid
                        gridType="circle"
                        radialLines={false}
                        stroke="#e5e7eb"
                        className="first:fill-muted last:fill-background"
                    />
                    <RadialBar dataKey="value" cornerRadius={10} />
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
                                                y={(viewBox.cy ?? 0) - 12}
                                                className="fill-foreground text-4xl font-bold"
                                            >
                                                {overallMeanPercentage.toFixed(1)}%
                                            </tspan>
                                            <tspan
                                                x={viewBox.cx}
                                                y={(viewBox.cy ?? 0) + 8}
                                                className="fill-muted-foreground text-xs"
                                            >
                                                {levelDescription(mappedPerformanceLevel)}
                                            </tspan>
                                            <tspan
                                                x={viewBox.cx}
                                                y={(viewBox.cy ?? 0) + 24}
                                                className="fill-muted-foreground text-[10px]"
                                            >
                                                {mappedPerformanceLevel}
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
