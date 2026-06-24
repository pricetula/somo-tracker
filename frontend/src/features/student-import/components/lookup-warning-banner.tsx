/**
 * Warning banner for degraded lookup mode.
 *
 * Displayed when parent or class lookups fail, with a [Retry Lookup] action.
 */

"use client";

import { AlertCircle, RefreshCw } from "lucide-react";

interface LookupWarningBannerProps {
    type: "parents" | "classes";
    message: string;
    onRetry: () => void;
}

export function LookupWarningBanner({ type, message, onRetry }: LookupWarningBannerProps) {
    const label = type === "parents" ? "Parent linking" : "Class linking";

    return (
        <div className="bg-muted/30 flex items-center justify-between px-3 py-2">
            <div className="flex items-center gap-2">
                <AlertCircle className="text-destructive size-4 shrink-0" />
                <p className="text-muted-foreground text-sm">
                    <span className="text-foreground font-medium">{label} unavailable</span>
                    &nbsp;&mdash; {message}
                </p>
            </div>
            <button
                onClick={onRetry}
                className="text-muted-foreground hover:text-foreground flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium"
            >
                <RefreshCw className="size-3" />
                Retry Lookup
            </button>
        </div>
    );
}
