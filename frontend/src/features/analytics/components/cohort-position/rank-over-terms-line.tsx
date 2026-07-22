/**
 * RankOverTermsLine — Line chart showing how class rank has changed across terms.
 *
 * Visualisation: Rank position over time — improving or slipping?
 * Props: Array of { termName, rank, headcount }.
 */
"use client";

import { CartesianGrid, Dot, Line, LineChart, XAxis, YAxis } from "recharts";

import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    rank: {
        label: "Class Rank",
        color: "#22c55e",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface RankOverTerm {
    termName: string;
    rank: number;
    headcount: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface RankOverTermsLineProps {
    data: RankOverTerm[];
}

export function RankOverTermsLine({ data }: RankOverTermsLineProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No rank data available.
            </p>
        );
    }

    const maxHeadcount = Math.max(...data.map((d) => d.headcount));

    // Invert Y axis so rank 1 is at the top
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Rank Trend Across Terms
                <GraphHelp>
                    Line chart showing how the student&rsquo;s class rank has changed across terms.
                    Rank 1 at the top means best-performing.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/1] w-full">
                <LineChart
                    accessibilityLayer
                    data={data}
                    margin={{ top: 8, left: 8, right: 8, bottom: 8 }}
                >
                    <CartesianGrid vertical={false} />
                    <XAxis dataKey="termName" tickLine={false} axisLine={false} tickMargin={8} />
                    <YAxis
                        reversed
                        domain={[1, maxHeadcount]}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        label={{
                            value: "Rank (1 = best)",
                            angle: -90,
                            position: "left",
                            style: { fontSize: 10, fill: "#6b7280" },
                        }}
                    />
                    <ChartTooltip
                        cursor={false}
                        content={
                            <ChartTooltipContent
                                formatter={(val, name, _item) => {
                                    const payload = (_item as unknown as Record<string, unknown>)
                                        ?.payload as RankOverTerm | undefined;
                                    if (name === "rank" && payload) {
                                        return `#${val} of ${payload.headcount}`;
                                    }
                                    return val;
                                }}
                            />
                        }
                    />
                    <Line
                        dataKey="rank"
                        type="monotone"
                        stroke="#22c55e"
                        strokeWidth={2}
                        dot={({ payload }) => <Dot key={payload.termName} r={5} fill="#22c55e" />}
                    />
                </LineChart>
            </ChartContainer>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function RankOverTermsLineSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
