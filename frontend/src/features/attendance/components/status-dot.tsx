"use client";

import { cn } from "@/lib/utils";

interface StatusDotProps {
    status: DayStatus;
}
export type DayStatus = "none" | "green" | "yellow" | "red";

export function StatusDot({ status }: StatusDotProps) {
    if (status === "none") return null;

    const colorClass = {
        green: "bg-green-500",
        yellow: "bg-yellow-500",
        red: "bg-red-500",
    }[status];

    return (
        <span
            className={cn(
                "absolute -bottom-0.5 left-1/2 h-1.5 w-1.5 -translate-x-1/2 rounded-full",
                colorClass
            )}
            aria-hidden="true"
        />
    );
}
