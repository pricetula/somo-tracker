/**
 * BehaviorCalendarHeatmap — When in the term do incidents cluster? Day/month grid.
 * Props: Array of { date, count }.
 */
"use client";

import { cn } from "@/lib/utils";
import { GraphHelp } from "@/features/analytics/components/graph-help";

export interface IncidentDay {
    date: string;
    count: number;
    dateLabel: string;
}

interface Props {
    data: IncidentDay[];
}

function cellColor(count: number): string {
    if (count === 0) return "bg-muted/20";
    if (count === 1) return "bg-amber-500/30";
    if (count === 2) return "bg-orange-500/50";
    return "bg-destructive/60";
}

export function BehaviorCalendarHeatmap({ data }: Props) {
    if (!data.length)
        return <p className="text-muted-foreground py-8 text-center text-sm">No incident data.</p>;

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Incident Calendar
                <GraphHelp>
                    Calendar heatmap showing when behaviour incidents occur throughout the term.
                    Darker cells indicate more incidents.
                </GraphHelp>
            </p>
            <div className="grid grid-cols-7 gap-1">
                {["M", "T", "W", "T", "F", "S", "S"].map((h) => (
                    <div key={h} className="text-muted-foreground h-5 text-center text-[10px]">
                        {h}
                    </div>
                ))}
                {data.map((d) => (
                    <div
                        key={d.date}
                        className={cn(
                            "flex h-6 w-8 items-center justify-center rounded text-[10px]",
                            cellColor(d.count)
                        )}
                        title={`${d.dateLabel}: ${d.count} incident${d.count !== 1 ? "s" : ""}`}
                    >
                        {d.count > 0 ? d.count : ""}
                    </div>
                ))}
            </div>
            <div className="flex items-center gap-3 pt-1">
                <span className="text-muted-foreground text-xs">Incidents:</span>
                {[
                    { label: "0", cls: "bg-muted/20" },
                    { label: "1", cls: "bg-amber-500/30" },
                    { label: "2", cls: "bg-orange-500/50" },
                    { label: `≥3`, cls: "bg-destructive/60" },
                ].map((e) => (
                    <div key={e.label} className="flex items-center gap-1">
                        <div className={cn("h-3 w-3 rounded", e.cls)} />
                        <span className="text-muted-foreground text-[10px]">{e.label}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}
