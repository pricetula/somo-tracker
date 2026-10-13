"use client";

import { Check } from "lucide-react";
import { Progress } from "@/components/ui/progress";

type Step = { id: string; label: string; done: boolean };

export function ProgressTracker({ steps }: { steps: Step[] }) {
    const completed = steps.filter((s) => s.done).length;
    const pct = steps.length ? (completed / steps.length) * 100 : 0;

    return (
        <div className="space-y-3">
            <Progress value={pct} />
            <div className="flex items-center gap-4 text-sm">
                {steps.map((s) => (
                    <div key={s.id} className="flex items-center gap-2">
                        <span
                            className={`flex size-5 items-center justify-center rounded-full border ${s.done ? "bg-primary text-primary-foreground border-primary" : "border-muted-foreground"}`}
                        >
                            {s.done ? <Check className="size-3" /> : " "}
                        </span>
                        <span className={s.done ? "font-medium" : "text-muted-foreground"}>
                            {s.label}
                        </span>
                    </div>
                ))}
            </div>
        </div>
    );
}
