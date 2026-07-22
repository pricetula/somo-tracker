/**
 * SubjectDotPlot — Dot plot with threshold lines showing EE/ME/AE/BE zones.
 *
 * Visualisation: Each subject as a dot on a horizontal scale with level bands.
 * Props: Array of { subject, score } entries with configurable thresholds.
 */
"use client";

import {
    CartesianGrid,
    Scatter,
    ScatterChart,
    XAxis,
    YAxis,
    ZAxis,
    ReferenceLine,
    ReferenceArea,
} from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    score: {
        label: "Score",
        color: "#1f2937",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubjectDotEntry {
    subject: string;
    score: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface SubjectDotPlotProps {
    data: SubjectDotEntry[];
}

export function SubjectDotPlot({ data }: SubjectDotPlotProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No dot plot data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Subject Scores with Level Thresholds
                <GraphHelp>
                    Dot plot with threshold bands showing each subject&rsquo;s score on a horizontal
                    scale with EE/ME/AE/BE level zones for quick visual assessment.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <ScatterChart margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid horizontal={false} strokeDasharray="3 3" />
                    <XAxis
                        dataKey="score"
                        type="number"
                        domain={[0, 100]}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                    />
                    <YAxis
                        dataKey="subject"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={110}
                    />
                    <ZAxis range={[100, 100]} />

                    {/* Threshold bands */}
                    <ReferenceArea x1={80} x2={100} fill="#22c55e" fillOpacity={0.08} />
                    <ReferenceArea x1={60} x2={80} fill="#3b82f6" fillOpacity={0.08} />
                    <ReferenceArea x1={40} x2={60} fill="#f59e0b" fillOpacity={0.08} />
                    <ReferenceArea x1={0} x2={40} fill="#ef4444" fillOpacity={0.08} />

                    {/* Threshold lines */}
                    <ReferenceLine x={80} stroke="#22c55e" strokeDasharray="4 4" strokeWidth={1} />
                    <ReferenceLine x={60} stroke="#3b82f6" strokeDasharray="4 4" strokeWidth={1} />
                    <ReferenceLine x={40} stroke="#f59e0b" strokeDasharray="4 4" strokeWidth={1} />

                    {/* Band labels via reference lines with label */}
                    <ReferenceLine
                        x={90}
                        strokeWidth={0}
                        label={{
                            value: "EE",
                            position: "top",
                            fill: "#6b7280",
                            fontSize: 10,
                        }}
                    />
                    <ReferenceLine
                        x={70}
                        strokeWidth={0}
                        label={{
                            value: "ME",
                            position: "top",
                            fill: "#6b7280",
                            fontSize: 10,
                        }}
                    />
                    <ReferenceLine
                        x={50}
                        strokeWidth={0}
                        label={{
                            value: "AE",
                            position: "top",
                            fill: "#6b7280",
                            fontSize: 10,
                        }}
                    />
                    <ReferenceLine
                        x={20}
                        strokeWidth={0}
                        label={{
                            value: "BE",
                            position: "top",
                            fill: "#6b7280",
                            fontSize: 10,
                        }}
                    />

                    <ChartTooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        content={
                            <ChartTooltipContent
                                formatter={(value) => {
                                    if (value == null || typeof value !== "number") return "";
                                    return `${value.toFixed(1)}%`;
                                }}
                            />
                        }
                    />
                    <Scatter data={data} fill="#1f2937" />
                </ScatterChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function SubjectDotPlotSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-52 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
