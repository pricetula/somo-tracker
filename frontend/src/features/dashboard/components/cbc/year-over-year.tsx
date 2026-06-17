"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Legend } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { YoYRow } from "./types";

export function YearOverYear({ data, hasPriorYear }: { data: YoYRow[]; hasPriorYear: boolean }) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2 text-xs">
                    Year-over-Year Comparison
                    {!hasPriorYear && (
                        <span className="text-muted-foreground font-normal">
                            No comparison data for 2025
                        </span>
                    )}
                </CardTitle>
            </CardHeader>
            <CardContent>
                <LineChart width={320} height={160} data={data}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="week" tick={{ fontSize: 10 }} />
                    <YAxis domain={[1, 4]} ticks={[1, 2, 3, 4]} tick={{ fontSize: 10 }} />
                    <Line
                        type="monotone"
                        dataKey="thisYear"
                        stroke="#7F77DD"
                        strokeWidth={2}
                        dot={false}
                        name="This Year"
                    />
                    {hasPriorYear && (
                        <Line
                            type="monotone"
                            dataKey="lastYear"
                            stroke="#B4B2A9"
                            strokeWidth={1.5}
                            strokeDasharray="4 4"
                            dot={false}
                            name="Last Year"
                        />
                    )}
                    <Legend wrapperStyle={{ fontSize: 10 }} iconType="line" />
                </LineChart>
            </CardContent>
        </Card>
    );
}
