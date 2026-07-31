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
import { GraphHelp } from "@/features/analytics/components/graph-help";

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
            return "#22c55e";
        case "ME":
            return "#3b82f6";
        case "AE":
            return "#f59e0b";
        case "BE":
            return "#ef4444";
        default:
            return "#f3f4f6";
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
            <p className="text-foreground text-sm font-medium">
                Subject Contribution
                <GraphHelp>
                    Treemap where rectangle size represents a subject&rsquo;s score contribution and
                    colour indicates performance level (EE/ME/AE/BE).
                </GraphHelp>
            </p>
            <p className="text-muted-foreground text-xs">
                Rectangle size = score, colour = performance level
            </p>
            <ChartContainer config={chartConfig} className="aspect-[4/3] w-full">
                <RechartsTreemap
                    data={data}
                    dataKey="size"
                    aspectRatio={4 / 3}
                    stroke="#ffffff"
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
                                            fill="#ffffff"
                                            fontSize={12}
                                            fontWeight={600}
                                        >
                                            {name}
                                        </text>
                                        <text
                                            x={x + width / 2}
                                            y={y + height / 2 + 14}
                                            textAnchor="middle"
                                            fill="#ffffff"
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
                    { label: "EE", cls: "bg-[#22c55e]" },
                    { label: "ME", cls: "bg-[#3b82f6]" },
                    { label: "AE", cls: "bg-[#f59e0b]" },
                    { label: "BE", cls: "bg-[#ef4444]" },
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
