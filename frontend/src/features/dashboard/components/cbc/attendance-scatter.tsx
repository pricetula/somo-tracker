"use client";

import { ScatterChart, Scatter, XAxis, YAxis, ReferenceLine } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { ScatterPoint } from "./types";

function getColor(score: number): string {
    if (score >= 3.5) return "#1D9E75";
    if (score >= 2.5) return "#3B82F6";
    if (score >= 1.5) return "#EF9F27";
    return "#E24B4A";
}

export function AttendanceScatter({ data }: { data: ScatterPoint[] }) {
    if (data.length < 5) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="text-xs">Attendance vs CBC Score</CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="text-muted-foreground flex h-[180px] items-center justify-center text-xs">
                        Not enough data to plot
                    </div>
                </CardContent>
            </Card>
        );
    }

    const coloredData = data.map((d) => ({ ...d, fill: getColor(d.score) }));

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">Attendance vs CBC Score</CardTitle>
            </CardHeader>
            <CardContent>
                <ScatterChart width={320} height={180}>
                    <XAxis
                        dataKey="attendance"
                        type="number"
                        domain={[40, 100]}
                        tick={{ fontSize: 10 }}
                        label={{
                            value: "Attendance %",
                            position: "bottom",
                            offset: -5,
                            style: { fontSize: 10 },
                        }}
                    />
                    <YAxis dataKey="score" type="number" domain={[1, 4]} tick={{ fontSize: 10 }} />
                    <ReferenceLine
                        x={75}
                        stroke="#B4B2A9"
                        strokeDasharray="4 4"
                        label={{ value: "75%", fontSize: 9, position: "top" }}
                    />
                    <ReferenceLine y={2.5} stroke="#EF9F27" strokeDasharray="4 4" />
                    <Scatter
                        data={coloredData}
                        fill="#1D9E75"
                        shape={(props: { cx?: number; cy?: number; fill?: string }) => {
                            const { cx, cy, fill } = props;
                            return <circle cx={cx} cy={cy} r={3} fill={fill} />;
                        }}
                    />
                </ScatterChart>
            </CardContent>
        </Card>
    );
}
