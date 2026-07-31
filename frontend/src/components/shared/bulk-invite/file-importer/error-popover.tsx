"use client";

import { AlertCircle } from "lucide-react";

export function ErrorPopover({ errors }: { errors: string[] }) {
    if (errors.length === 0) return null;
    return (
        <div className="flex items-center gap-1">
            <AlertCircle className="text-destructive size-3 shrink-0" />
            <span className="text-destructive truncate text-xs">{errors[0]}</span>
            {errors.length > 1 && (
                <span className="text-muted-foreground text-[10px]">+{errors.length - 1}</span>
            )}
        </div>
    );
}
