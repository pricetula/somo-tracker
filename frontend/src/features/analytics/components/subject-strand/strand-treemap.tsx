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
            <p className="text-foreground text-sm font-medium">Strand Explorer — {data.name}</p>
            <ChartContainer config={chartConfig} className="aspect-[4/3] w-full">
                <RechartsTreemap
                    data={[data]}
                    dataKey="size"
                    aspectRatio={4 / 3}
                    stroke="hsl(var(--background))"
                    content={({ x, y, width, height, depth, name, payload }) => {
                        if (depth === 0) return React.createElement(React.Fragment, null);
                        const node = payload as SubStrandNode | StrandNode | undefined;
                        const fill =
                            depth === 1
                                ? "hsl(var(--muted))"
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
                                    stroke={
                                        depth === 2
                                            ? "hsl(var(--background))"
                                            : "hsl(var(--border))"
                                    }
                                    strokeWidth={depth === 1 ? 1 : 2}
                                />
                                {width > 30 && height > 20 && (
                                    <text
                                        x={x + width / 2}
                                        y={y + height / 2}
                                        textAnchor="middle"
                                        dominantBaseline="middle"
                                        fill={
                                            depth === 1
                                                ? "hsl(var(--muted-foreground))"
                                                : "hsl(var(--background))"
                                        }
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

export function StrandTreemapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-44 animate-pulse rounded" />
            <div className="bg-muted aspect-[4/3] w-full animate-pulse rounded" />
        </div>
    );
}
