"use client";

import { PieChart, Pie, Cell } from "recharts";
import { Card, CardContent } from "@/components/ui/card";
import type { AttendanceData } from "./types";

const COLORS = { present: "#1D9E75", late: "#EF9F27", absent: "#E24B4A" };

export function AttendanceSnapshot({ data }: { data: AttendanceData }) {
    if (data.total === 0) {
        return (
            <Card>
                <CardContent>
                    <div className="text-muted-foreground flex h-full items-center justify-center py-4 text-xs">
                        No attendance recorded today
                    </div>
                </CardContent>
            </Card>
        );
    }

    const presentRate = Math.round((data.present / data.total) * 100);
    const chartData = [
        { name: "present", value: data.present },
        { name: "late", value: data.late },
        { name: "absent", value: data.absent },
    ];

    return (
        <Card>
            <CardContent>
                <div className="flex flex-col items-center gap-2">
                    {/* Donut chart with center label */}
                    <div className="relative flex items-center justify-center">
                        <PieChart width={160} height={140}>
                            <Pie
                                data={chartData}
                                cx={75}
                                cy={70}
                                innerRadius={50}
                                outerRadius={70}
                                dataKey="value"
                                strokeWidth={0}
                            >
                                <Cell fill={COLORS.present} />
                                <Cell fill={COLORS.late} />
                                <Cell fill={COLORS.absent} />
                            </Pie>
                        </PieChart>
                        <span className="absolute text-lg font-bold">{presentRate}%</span>
                    </div>

                    {/* Inline pills */}
                    <div className="flex items-center gap-2 text-xs">
                        <span className="flex items-center gap-1">
                            <span
                                className="inline-block size-2 rounded-full"
                                style={{ backgroundColor: COLORS.present }}
                            />
                            {data.present} present
                        </span>
                        <span className="flex items-center gap-1">
                            <span
                                className="inline-block size-2 rounded-full"
                                style={{ backgroundColor: COLORS.late }}
                            />
                            {data.late} late
                        </span>
                        <span className="flex items-center gap-1">
                            <span
                                className="inline-block size-2 rounded-full"
                                style={{ backgroundColor: COLORS.absent }}
                            />
                            {data.absent} absent
                        </span>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
