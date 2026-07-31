"use client";

import { cn } from "@/lib/utils";

export interface MomentumData {
    momentumScore: number;
    subjectName?: string;
}
function momentumConfig(score: number) {
    if (score > 2) return { arrow: "\u2191", label: "Improving", cls: "text-emerald-600" };
    if (score < -2) return { arrow: "\u2193", label: "Declining", cls: "text-destructive" };
    return { arrow: "\u2192", label: "Stable", cls: "text-muted-foreground" };
}
interface MomentumArrowProps {
    data: MomentumData;
}

export function MomentumArrow({ data }: MomentumArrowProps) {
    const config = momentumConfig(data.momentumScore);

    return (
        <div className="flex items-center gap-2">
            {data.subjectName && (
                <span className="text-muted-foreground text-xs">{data.subjectName}</span>
            )}
            <span className={cn("text-lg font-bold", config.cls)}>{config.arrow}</span>
            <span className={cn("text-xs font-medium", config.cls)}>{config.label}</span>
            <span className={cn("text-xs tabular-nums", config.cls)}>
                ({data.momentumScore > 0 ? "+" : ""}
                {data.momentumScore.toFixed(2)})
            </span>
        </div>
    );
}
