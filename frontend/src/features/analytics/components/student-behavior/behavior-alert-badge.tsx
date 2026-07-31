/**
 * BehaviorAlertBadge — Alert badge for disciplinary incidents / urgent flags.
 */
"use client";

import { cn } from "@/lib/utils";

interface Props {
    totalIncidents: number;
    urgentCount: number;
    disciplinaryCount: number;
    variant?: "badge" | "card";
}

export function BehaviorAlertBadge({
    totalIncidents,
    urgentCount,
    disciplinaryCount,
    variant = "badge",
}: Props) {
    if (variant === "card") {
        return (
            <div
                className={cn(
                    "space-y-2 rounded-md p-3",
                    disciplinaryCount > 2 ? "bg-destructive/10" : "bg-amber-500/10"
                )}
            >
                <p
                    className={cn(
                        "text-sm font-medium",
                        disciplinaryCount > 2 ? "text-destructive" : "text-amber-600"
                    )}
                >
                    Behaviour Alert
                </p>
                <p className="text-foreground text-sm">
                    {disciplinaryCount} disciplinary incident{disciplinaryCount !== 1 ? "s" : ""}{" "}
                    this term
                    {urgentCount > 0 &&
                        ` — ${urgentCount} urgent flag${urgentCount !== 1 ? "s" : ""} raised`}
                </p>
                <p className="text-muted-foreground text-xs">{totalIncidents} total incidents</p>
            </div>
        );
    }

    if (disciplinaryCount === 0 && urgentCount === 0) return null;

    return (
        <span
            className={cn(
                "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                disciplinaryCount > 2
                    ? "bg-destructive/15 text-destructive"
                    : "bg-amber-500/15 text-amber-600"
            )}
        >
            <span className="h-1.5 w-1.5 rounded-full bg-current" />
            {disciplinaryCount > 0 ? `${disciplinaryCount} disciplinary` : `${urgentCount} urgent`}
        </span>
    );
}
