/**
 * DistributionCurve — Bell curve showing grade score distribution with student's position highlighted.
 *
 * Visualisation: Normal distribution of grade scores, with a marker for the student.
 * Props: { studentScore, gradeAverage, gradeStdDev?, dataPoints? }.
 */
"use client";

import { Area, AreaChart, CartesianGrid, ReferenceLine, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    density: {
        label: "Distribution",
        color: "#3b82f6",
    },
} satisfies ChartConfig;

// ─── Helpers ──────────────────────────────────────────────────────────────

/** Generate points for a normal distribution curve */
function generateNormalCurve(
    mean: number,
    stdDev: number,
    pointsCount = 40
): { score: number; density: number }[] {
    const points: { score: number; density: number }[] = [];
    const min = Math.max(0, mean - 3.5 * stdDev);
    const max = Math.min(100, mean + 3.5 * stdDev);
    const step = (max - min) / pointsCount;

    for (let i = 0; i <= pointsCount; i++) {
        const score = min + i * step;
        const density =
            (1 / (stdDev * Math.sqrt(2 * Math.PI))) *
            Math.exp(-0.5 * ((score - mean) / stdDev) ** 2);
        points.push({ score: Math.round(score * 10) / 10, density });
    }
    return points;
}

// ─── Types ────────────────────────────────────────────────────────────────

export interface DistributionCurveData {
    studentScore: number;
    gradeAverage: number;
    gradeStdDev?: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface DistributionCurveProps {
    data: DistributionCurveData;
}

export function DistributionCurve({ data }: DistributionCurveProps) {
    const { studentScore, gradeAverage, gradeStdDev = 10 } = data;
    const curve = generateNormalCurve(gradeAverage, gradeStdDev);

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Grade Distribution
                <GraphHelp>
                    Bell curve showing the grade&rsquo;s score distribution with the student&rsquo;s
                    position highlighted. Helps visualise where the student stands relative to
                    peers.
                </GraphHelp>
            </p>
            <p className="text-muted-foreground text-xs">
                Your score highlighted on the grade curve
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <AreaChart data={curve} margin={{ top: 8, left: 8, right: 8, bottom: 8 }}>
                    <CartesianGrid vertical={false} strokeDasharray="3 3" />
                    <XAxis
                        dataKey="score"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        domain={[0, 100]}
                    />
                    <YAxis hide />
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                formatter={(val: unknown) => {
                                    const v = Number(val);
                                    return isNaN(v) ? "" : v.toFixed(4);
                                }}
                            />
                        }
                    />
                    {/* Distribution area */}
                    <Area
                        type="monotone"
                        dataKey="density"
                        stroke="#3b82f6"
                        fill="#3b82f6"
                        fillOpacity={0.15}
                        strokeWidth={2}
                    />
                    {/* Grade average line */}
                    <ReferenceLine
                        x={gradeAverage}
                        stroke="#6b7280"
                        strokeDasharray="4 4"
                        label={{
                            value: "Avg",
                            position: "top",
                            fill: "#6b7280",
                            fontSize: 10,
                        }}
                    />
                    {/* Student score marker */}
                    <ReferenceLine
                        x={studentScore}
                        stroke="#22c55e"
                        strokeWidth={2}
                        label={{
                            value: "You",
                            position: "top",
                            fill: "#22c55e",
                            fontSize: 11,
                            fontWeight: 600,
                        }}
                    />
                </AreaChart>
            </ChartContainer>

            <div className="flex items-center justify-center gap-6 text-xs">
                <div className="flex items-center gap-1">
                    <div className="h-2 w-4 rounded bg-[#3b82f6]" />
                    <span className="text-muted-foreground">Grade distribution</span>
                </div>
                <div className="flex items-center gap-1">
                    <div className="bg-muted-foreground h-0.5 w-4" />
                    <span className="text-muted-foreground">
                        Grade avg: {gradeAverage.toFixed(1)}%
                    </span>
                </div>
                <div className="flex items-center gap-1">
                    <div className="h-2 w-2 rounded-full bg-[#22c55e]" />
                    <span className="text-muted-foreground">You: {studentScore.toFixed(1)}%</span>
                </div>
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
