/**
 * StrandTreemap — Hierarchical treemap: Subject → Strand → Sub-Strand drill-down.
 *
 * Visualisation: Nested rectangles coloured by performance level.
 * Props: Hierarchical data structure.
 */
"use client";

import React from "react";
import { Treemap as RechartsTreemap } from "recharts";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Types ────────────────────────────────────────────────────────────────

export interface SubStrandNode {
    [key: string]: unknown;
    name: string;
    size: number;
    level: string;
}

export interface StrandNode {
    [key: string]: unknown;
    name: string;
    children: SubStrandNode[];
}

export interface StrandTreemapHierarchy {
    [key: string]: unknown;
    name: string;
    children: StrandNode[];
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

// ─── Config ───────────────────────────────────────────────────────────────

const chartConfig = {
    size: { label: "Mastery" },
} satisfies ChartConfig;

// ─── Component ────────────────────────────────────────────────────────────

interface StrandTreemapProps {
    data: StrandTreemapHierarchy;
}

export function StrandTreemap({ data }: StrandTreemapProps) {
    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Strand Explorer — {data.name}
                <GraphHelp>
                    Hierarchical treemap drill-down from Subject → Strand → Sub-Strand, coloured by
                    performance level for easy visual exploration.
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[4/3] w-full">
                <RechartsTreemap
                    data={[data]}
                    dataKey="size"
                    aspectRatio={4 / 3}
                    stroke="#ffffff"
                    content={({ x, y, width, height, depth, name, payload }) => {
                        if (depth === 0) return React.createElement(React.Fragment, null);
                        const node = payload as SubStrandNode | StrandNode | undefined;
                        const fill =
                            depth === 1
                                ? "#f3f4f6"
                                : levelFill((node as SubStrandNode)?.level ?? "AE");

                        return (
                            <g>
                                <rect
                                    x={x}
                                    y={y}
                                    width={width}
                                    height={height}
                                    fill={fill}
                                    rx={depth === 2 ? 2 : 4}
                                    ry={depth === 2 ? 2 : 4}
                                    stroke={depth === 2 ? "#ffffff" : "#e5e7eb"}
                                    strokeWidth={depth === 1 ? 1 : 2}
                                />
                                {width > 30 && height > 20 && (
                                    <text
                                        x={x + width / 2}
                                        y={y + height / 2}
                                        textAnchor="middle"
                                        dominantBaseline="middle"
                                        fill={depth === 1 ? "#6b7280" : "#ffffff"}
                                        fontSize={depth === 1 ? 11 : 10}
                                        fontWeight={depth === 1 ? 500 : 600}
                                    >
                                        {name}
                                    </text>
                                )}
                            </g>
                        );
                    }}
                />
            </ChartContainer>

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
