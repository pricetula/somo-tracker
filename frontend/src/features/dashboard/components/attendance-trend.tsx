"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { Line, LineChart, XAxis, YAxis, CartesianGrid } from "recharts";

const data = [
    { date: "Mon", attendancePct: 92, markingPct: 88 },
    { date: "Tue", attendancePct: 94, markingPct: 75 },
    { date: "Wed", attendancePct: 90, markingPct: 82 },
    { date: "Thu", attendancePct: 93, markingPct: 91 },
    { date: "Fri", attendancePct: 91, markingPct: 87 },
    { date: "Sat", attendancePct: 0, markingPct: 0 },
    { date: "Sun", attendancePct: 0, markingPct: 0 },
];

const config = {
    attendancePct: {
        label: "Attendance %",
        theme: { light: "hsl(142 70% 45%)", dark: "hsl(142 70% 45%)" },
    },
    markingPct: {
        label: "Marking complete %",
        theme: { light: "hsl(221 83% 53%)", dark: "hsl(221 83% 53%)" },
    },
};

export function AttendanceTrend() {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-sm">ATTENDANCE TREND · Last 7 days</CardTitle>
            </CardHeader>
            <CardContent>
                <ChartContainer config={config}>
                    <LineChart data={data} margin={{ left: -20, right: 10, top: 10, bottom: 0 }}>
                        <CartesianGrid vertical={false} />
                        <XAxis dataKey="date" tickLine={false} axisLine={false} />
                        <YAxis domain={[0, 100]} tickLine={false} axisLine={false} width={30} />
                        <ChartTooltip content={<ChartTooltipContent />} />
                        <Line
                            type="monotone"
                            dataKey="attendancePct"
                            stroke="var(--color-attendancePct)"
                            strokeWidth={2}
                            dot={false}
                        />
                        <Line
                            type="monotone"
                            dataKey="markingPct"
                            stroke="var(--color-markingPct)"
                            strokeWidth={2}
                            dot={false}
                        />
                    </LineChart>
                </ChartContainer>
            </CardContent>
        </Card>
    );
}
