"use client";

import { Check, X, AlertTriangle } from "lucide-react";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";

type Step = { id: string; label: string; done: boolean };

export function ProgressTracker({ steps }: { steps: Step[] }) {
    const completed = steps.filter((s) => s.done).length;
    const total = steps.length;
    const pct = total ? (completed / total) * 100 : 0;

    const status = completed === 0 ? "none" : completed === total ? "done" : "semi";

    const Icon = status === "done" ? Check : status === "semi" ? AlertTriangle : X;
    const iconColor =
        status === "done"
            ? "text-emerald-600"
            : status === "semi"
              ? "text-primary"
              : "text-destructive";

    const radius = 15.9155;
    const circumference = 2 * Math.PI * radius;
    const offset = circumference - (pct / 100) * circumference;

    return (
        <div className="space-y-3">
            <Popover>
                <PopoverTrigger
                    className="focus-visible:ring-ring/30 relative flex size-6 items-center justify-center rounded-full p-0 shadow-none hover:bg-transparent focus-visible:ring-2 focus-visible:ring-offset-0"
                    aria-label="Setup progress"
                >
                    <svg
                        className="absolute inset-0 size-full -rotate-90"
                        viewBox="0 0 36 36"
                        aria-hidden="true"
                    >
                        <circle
                            cx="18"
                            cy="18"
                            r={radius}
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="3"
                            strokeDasharray={circumference}
                            strokeDashoffset={offset}
                            className="text-primary"
                        />
                    </svg>
                    <span className={`relative z-10 ${iconColor}`}>
                        <Icon className="size-3" />
                    </span>
                </PopoverTrigger>
                <PopoverContent align="start" className="w-64 gap-2">
                    <div className="text-muted-foreground text-xs font-medium">Setup steps</div>
                    <div className="space-y-2">
                        {steps.map((s) => (
                            <div
                                key={s.id}
                                className={`flex items-center gap-2 text-sm ${s.done ? "text-muted-foreground line-through" : "text-foreground"}`}
                            >
                                <span
                                    className={`inline-block size-2 rounded-full ${s.done ? "bg-emerald-600" : "bg-muted-foreground"}`}
                                />
                                <span>{s.label}</span>
                            </div>
                        ))}
                    </div>
                </PopoverContent>
            </Popover>
            <div className="text-muted-foreground flex items-center gap-2 text-xs">
                <span>{completed}</span>
                <span>/</span>
                <span>{total}</span>
                <span>completed</span>
            </div>
        </div>
    );
}
