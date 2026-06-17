"use client";

import { BarChart, Bar, XAxis, Legend } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { StreamRow } from "./types";

export function StreamPerformance({ data, streams }: { data: StreamRow[]; streams: string[] }) {
    const singleStream = streams.length <= 1;
    const headerLabel = singleStream
        ? "Single stream — no comparison available"
        : "Stream Performance Variance";

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">{headerLabel}</CardTitle>
            </CardHeader>
            <CardContent>
                <BarChart width={320} height={180} data={data} barCategoryGap="30%" barGap={2}>
                    <XAxis dataKey="level" tick={{ fontSize: 11 }} />
                    {streams.length >= 1 && (
                        <Bar dataKey="east" fill="#1D9E75" name={streams[0] ?? "East"} />
                    )}
                    {streams.length >= 2 && (
                        <Bar dataKey="west" fill="#7F77DD" name={streams[1] ?? "West"} />
                    )}
                    <Legend wrapperStyle={{ fontSize: 10 }} iconType="rect" iconSize={8} />
                </BarChart>
            </CardContent>
        </Card>
    );
}
