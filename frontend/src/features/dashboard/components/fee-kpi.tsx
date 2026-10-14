"use client";

import { ChartContainer } from "@/components/ui/chart";
import { Line, LineChart, XAxis, Tooltip } from "recharts";

const months = ["Month 1", "Month 2", "Month 3"];
const currentTerm = [80, 135, 160];
const lastTerm = [75, 125, 148];
const target = [150, 150, 150];

const data = months.map((m, i) => ({
    month: m,
    c: currentTerm[i],
    l: lastTerm[i],
    t: target[i],
}));

const config = {
    c: { label: "Current term", theme: { light: "hsl(221 83% 53%)", dark: "hsl(221 83% 53%)" } },
    l: { label: "Last term", theme: { light: "hsl(142 70% 45%)", dark: "hsl(142 70% 45%)" } },
    t: { label: "Target", theme: { light: "hsl(0 0% 60%)", dark: "hsl(0 0% 60%)" } },
};

function CustomTooltip({
    active,
    payload,
}: {
    active?: boolean;
    payload?: { dataKey: string; value: number; payload?: { month?: string } }[];
}) {
    if (!active || !payload?.length) return null;
    const month = (payload?.[0]?.payload as { month?: string })?.month;
    return (
        <div className="bg-background min-w-60 rounded border px-3 py-2 text-xs shadow">
            <div className="mb-1 font-medium">{month} collections</div>
            {payload?.map((p) => (
                <div key={p.dataKey} className="flex gap-2">
                    <span className="text-muted-foreground">
                        {config[p.dataKey as keyof typeof config]?.label}
                    </span>
                    <span>Ksh {String(p.value)}k</span>
                </div>
            ))}
        </div>
    );
}

export function FeeKpi() {
    const value = 68;
    return (
        <div className="flex items-center justify-between border p-3">
            <div>
                <div className="text-muted-foreground text-sm">Fees collected</div>
                <div className="text-2xl">{value}%</div>
                <div className="text-muted-foreground text-xs">38 students outstanding</div>
            </div>
            <div className="h-8 w-32">
                <ChartContainer config={config}>
                    <LineChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
                        <XAxis dataKey="month" hide />
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
                        <Line
                            type="monotone"
                            dataKey="t"
                            stroke="var(--color-t)"
                            strokeWidth={1.5}
                            dot={false}
                            strokeDasharray="4 4"
                        />
                    </LineChart>
                </ChartContainer>
            </div>
        </div>
    );
}
