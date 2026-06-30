import * as React from "react";
import { AlertCircle, AlertTriangle } from "lucide-react";

export function CellWrapper({
    hasError,
    advisory,
    children,
}: {
    hasError?: boolean;
    advisory?: boolean;
    children: React.ReactNode;
}) {
    return (
        <div className="relative">
            {children}
            {hasError && (
                <AlertCircle className="text-destructive absolute top-1/2 right-2 size-3.5 -translate-y-1/2" />
            )}
            {advisory && !hasError && (
                <AlertTriangle className="absolute top-1/2 right-2 size-3.5 -translate-y-1/2 text-emerald-600" />
            )}
        </div>
    );
}
