/**
 * SubjectTreemap — Treemap showing subject contribution to overall, colour-coded by level.
 *
 * Visualisation: Hierarchical rectangles sized by score, coloured by level.
 * Props: Array of { name, size, level } entries.
 */
"use client";

import React from "react";
import { Treemap as RechartsTreemap } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    size: {
        label: "Score",
    },
} satisfies ChartConfig;

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubjectTreemapEntry {
    [key: string]: unknown;
    name: string;
    size: number;
    level: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function levelFill(level: string): string {
    switch (level) {
        case "EE":
            return "hsl(var(--chart-2))";
        case "ME":
            return "hsl(var(--chart-3))";
        case "AE":
            return "hsl(var(--chart-4))";
        case "BE":
            return "hsl(var(--chart-1))";
        default:
            return "hsl(var(--muted))";
    }
}

// ─── Component ────────────────────────────────────────────────────────────

interface SubjectTreemapProps {
    data: SubjectTreemapEntry[];
}

export function SubjectTreemap({ data }: SubjectTreemapProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No subject contribution data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Subject Contribution</p>
            <p className="text-muted-foreground text-xs">
                Rectangle size = score, colour = performance level
            </p>
            <ChartContainer config={chartConfig} className="aspect-[4/3] w-full">
                <RechartsTreemap
                    data={data}
                    dataKey="size"
                    aspectRatio={4 / 3}
                    stroke="hsl(var(--background))"
                    content={({ x, y, width, height, depth, name, payload }) => {
                        if (depth !== 1) return React.createElement(React.Fragment, null);
                        const lvl = (payload as SubjectTreemapEntry)?.level ?? "AE";
                        const fill = levelFill(lvl);
                        return (
                            <g>
                                <rect
                                    x={x}
                                    y={y}
                                    width={width}
                                    height={height}
                                    fill={fill}
                                    rx={4}
                                    ry={4}
                                    style={{ cursor: "pointer" }}
                                />
                                {width > 40 && height > 30 && (
                                    <>
                                        <text
                                            x={x + width / 2}
                                            y={y + height / 2 - 4}
                                            textAnchor="middle"
                                            fill="hsl(var(--background))"
                                            fontSize={12}
                                            fontWeight={600}
                                        >
                                            {name}
                                        </text>
                                        <text
                                            x={x + width / 2}
                                            y={y + height / 2 + 14}
                                            textAnchor="middle"
                                            fill="hsl(var(--background))"
                                            fontSize={11}
                                            opacity={0.9}
                                        >
                                            {lvl}
                                        </text>
                                    </>
                                )}
                            </g>
                        );
                    }}
                />
            </ChartContainer>

            {/* Level legend */}
            <div className="flex items-center gap-4">
                <span className="text-muted-foreground text-xs">Levels:</span>
                {[
                    { label: "EE", cls: "bg-[hsl(var(--chart-2))]" },
                    { label: "ME", cls: "bg-[hsl(var(--chart-3))]" },
                    { label: "AE", cls: "bg-[hsl(var(--chart-4))]" },
                    { label: "BE", cls: "bg-[hsl(var(--chart-1))]" },
                ].map((entry) => (
                    <div key={entry.label} className="flex items-center gap-1">
                        <div className={entry.cls + " h-3 w-3 rounded"} />
                        <span className="text-muted-foreground text-xs">{entry.label}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function SubjectTreemapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted aspect-[4/3] w-full animate-pulse rounded" />
        </div>
    );
}
