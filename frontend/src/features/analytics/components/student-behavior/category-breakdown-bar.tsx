/**
 * CategoryBreakdownBar — Each behaviour category with its count (horizontal bar).
 * Props: Array of { categoryName, categoryType, count }.
 */
"use client";

import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";
import {
    type ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { GraphHelp } from "@/features/analytics/components/graph-help";

export interface CategoryCountEntry {
    categoryName: string;
    categoryType: string;
    count: number;
}

interface Props {
    data: CategoryCountEntry[];
}

export function CategoryBreakdownBar({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No category data.</p>;
    const sorted = [...data].sort((a, b) => b.count - a.count);
    const chartConfig: ChartConfig = {};
    sorted.forEach((e) => {
        chartConfig[e.categoryName] = { label: e.categoryName };
    });

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Behaviour Categories
                <GraphHelp>
                    Horizontal bar chart showing counts for each behaviour category, colour-coded by
                    type (green for commendations, red for disciplinary).
                </GraphHelp>
            </p>
            <ChartContainer config={chartConfig} className="aspect-[3/2] w-full">
                <BarChart data={sorted} layout="vertical" barCategoryGap="20%">
                    <CartesianGrid horizontal={false} />
                    <XAxis type="number" tickLine={false} axisLine={false} allowDecimals={false} />
                    <YAxis
                        dataKey="categoryName"
                        type="category"
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        width={130}
                        tick={{ fontSize: 11 }}
                    />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
                    <Bar dataKey="count" radius={[0, 4, 4, 0]} barSize={18}>
                        {sorted.map((e) => (
                            <Cell
                                key={e.categoryName}
                                fill={e.categoryType === "COMMENDATION" ? "#22c55e" : "#ef4444"}
                            />
                        ))}
                    </Bar>
                </BarChart>
            </ChartContainer>
        </div>
    );
}

export function CategoryBreakdownBarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-48 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/2] w-full animate-pulse rounded" />
        </div>
    );
}
