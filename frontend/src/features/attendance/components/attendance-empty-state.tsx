/**
 * AttendanceEmptyState — reusable empty state for attendance views.
 * Pure shadcn: no borders, no cards, no hardcoded colours.
 */

import type { ReactNode } from "react";
import { type LucideIcon } from "lucide-react";

interface AttendanceEmptyStateProps {
    icon: LucideIcon;
    title: string;
    description?: string;
    children?: ReactNode;
}

export function AttendanceEmptyState({
    icon: Icon,
    title,
    description,
    children,
}: AttendanceEmptyStateProps) {
    return (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
            <div className="bg-muted flex size-16 items-center justify-center rounded-full">
                <Icon className="text-muted-foreground size-8" />
            </div>
            <div className="max-w-sm space-y-1">
                <p className="text-foreground font-semibold">{title}</p>
                {description && <p className="text-muted-foreground">{description}</p>}
            </div>
            {children && <div className="pt-2">{children}</div>}
        </div>
    );
}
