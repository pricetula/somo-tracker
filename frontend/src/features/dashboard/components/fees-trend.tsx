"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { Area, AreaChart, XAxis, YAxis, CartesianGrid } from "recharts";

const data = [
    { week: "W1", collected: 120000, target: 150000 },
    { week: "W2", collected: 135000, target: 150000 },
    { week: "W3", collected: 145000, target: 150000 },
    { week: "W4", collected: 128000, target: 150000 },
    { week: "W5", collected: 142000, target: 150000 },
    { week: "W6", collected: 155000, target: 150000 },
    { week: "W7", collected: 168000, target: 150000 },
    { week: "W8", collected: 162000, target: 150000 },
];

const config = {
    collected: {
        label: "Collected",
        theme: { light: "hsl(221 83% 53%)", dark: "hsl(221 83% 53%)" },
    },
    target: {
        label: "Target",
        theme: { light: "hsl(0 0% 60%)", dark: "hsl(0 0% 60%)" },
    },
};

export function FeesTrend() {
    const aging = [
        { label: "0-30d", value: "62%" },
        { label: "31-60d", value: "25%" },
        { label: "60d+", value: "13%" },
    ];

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-sm">FEES COLLECTION · Term 3</CardTitle>
            </CardHeader>
            <CardContent>
                <ChartContainer config={config}>
                    <AreaChart data={data} margin={{ left: -20, right: 10, top: 10, bottom: 0 }}>
                        <CartesianGrid vertical={false} />
                        <XAxis dataKey="week" tickLine={false} axisLine={false} />
                        <YAxis
                            tickLine={false}
                            axisLine={false}
                            width={50}
                            tickFormatter={(v) => `Ksh ${v / 1000}k`}
                        />
                        <ChartTooltip content={<ChartTooltipContent />} />
                        <Area
                            type="monotone"
                            dataKey="collected"
                            stroke="var(--color-collected)"
                            fill="var(--color-collected)"
                            fillOpacity={0.2}
                        />
                        <Area
                            type="monotone"
                            dataKey="target"
                            stroke="var(--color-target)"
                            fill="none"
                            strokeDasharray="4 4"
                        />
                    </AreaChart>
                </ChartContainer>
                <div className="text-muted-foreground mt-3 flex items-center justify-between text-xs">
                    <span>Aging</span>
                    <div className="flex gap-3">
                        {aging.map((a) => (
                            <span key={a.label} className="tabular-nums">
                                {a.label} {a.value}
                            </span>
                        ))}
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
