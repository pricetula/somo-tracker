"use client";

import { ChartContainer } from "@/components/ui/chart";
import { Line, LineChart, XAxis, Tooltip } from "recharts";
import { ChevronUp, ChevronDown } from "lucide-react";

const currentWeek = [{ v: 81 }, { v: 92 }, { v: 50 }, { v: 94 }, { v: 91 }];

const lastWeek = [{ v: 85 }, { v: 89 }, { v: 87 }, { v: 90 }, { v: 88 }];

const days = ["Mon", "Tue", "Wed", "Thu", "Fri"];
const data = currentWeek.map((c, i) => ({ day: days[i], c: c.v, l: lastWeek[i].v }));

const config = {
    c: { label: "Current", theme: { light: "hsl(142 70% 45%)", dark: "hsl(142 70% 45%)" } },
    l: { label: "Last week", theme: { light: "hsl(0 0% 60%)", dark: "hsl(0 0% 60%)" } },
};

function CustomTooltip({
    active,
    payload,
}: {
    active?: boolean;
    payload?: { dataKey: string; value: number; payload?: { day?: string } }[];
}) {
    if (!active || !payload?.length) return null;
    const day = (payload?.[0]?.payload as { day?: string })?.day;
    return (
        <div className="bg-background rounded border px-2 py-1 text-xs shadow">
            <div className="font-medium">{day} attendance %</div>
            {payload?.map((p) => (
                <div key={p.dataKey} className="flex gap-2">
                    <span className="text-muted-foreground">
                        {config[p.dataKey as keyof typeof config]?.label}
                    </span>
                    <span>{String(p.value)}%</span>
                </div>
            ))}
        </div>
    );
}

export function AttendanceKpi() {
    const value = 94;
    const change = 2;
    const isUp = change >= 0;
    return (
        <div className="flex items-center justify-between border p-3">
            <div>
                <div className="text-muted-foreground mb-4 text-sm">Attendance today</div>
                <div className="text-2xl">{value}%</div>
                <div className="text-muted-foreground flex items-center gap-1 text-xs">
                    {isUp ? (
                        <ChevronUp className="text-green-500" size={14} />
                    ) : (
                        <ChevronDown className="text-red-500" size={14} />
                    )}
                    {Math.abs(change)}% vs yesterday
                </div>
            </div>
            <div className="h-8 w-32">
                <ChartContainer config={config}>
                    <LineChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
                        <XAxis dataKey="day" hide />
                        <Tooltip content={<CustomTooltip />} />
                        <Line
                            type="monotone"
                            dataKey="c"
                            stroke="var(--color-c)"
                            strokeWidth={2}
                            dot={false}
                        />
                        <Line
                            type="monotone"
                            dataKey="l"
                            stroke="var(--color-l)"
                            strokeWidth={1.5}
                            dot={false}
                            strokeDasharray="3 3"
                        />
                    </LineChart>
                </ChartContainer>
            </div>
        </div>
    );
}
