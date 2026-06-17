"use client";

import { BarChart, Bar, XAxis, YAxis, Tooltip } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { MasteryRow } from "./types";

const COLORS = { EE: "#1D9E75", ME: "#9FE1CB", AE: "#EF9F27", BE: "#E24B4A" };
const LEVELS = ["EE", "ME", "AE", "BE"] as const;

export function LearningAreaMastery({ data }: { data: MasteryRow[] }) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">Learning Area Mastery</CardTitle>
            </CardHeader>
            <CardContent>
                <BarChart
                    width={320}
                    height={200}
                    data={data}
                    layout="vertical"
                    stackOffset="expand"
                >
                    <YAxis dataKey="area" type="category" width={90} tick={{ fontSize: 11 }} />
                    <XAxis hide type="number" />
                    <Tooltip
                        contentStyle={{ fontSize: 11 }}
                        formatter={(value: number | string) => `${value}%`}
                    />
                    {LEVELS.map((level) => (
                        <Bar key={level} dataKey={level} stackId="a" fill={COLORS[level]} />
                    ))}
                </BarChart>

                {/* Legend */}
                <div className="mt-2 flex items-center justify-center gap-3 text-[10px]">
                    {LEVELS.map((level) => (
                        <span key={level} className="flex items-center gap-1">
                            <span
                                className="inline-block size-2.5 rounded-sm"
                                style={{ backgroundColor: COLORS[level] }}
                            />
                            {level}
                        </span>
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}
