"use client";
import { cn } from "@/lib/utils";
import type { AttendanceStatus } from "../types/attendance";

const statusConfig: Record<AttendanceStatus, { label: string; className: string }> = {
    SUBMITTED: {
        label: "Submitted",
        className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400",
    },
    IN_PROGRESS: {
        label: "In Progress",
        className: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400",
    },
    MISSED: {
        label: "Missed",
        className: "bg-rose-100 text-rose-800 dark:bg-rose-900/30 dark:text-rose-400",
    },
    NO_STUDENTS: { label: "No Students", className: "bg-muted text-muted-foreground" },
};

export function StatusBadge({ status }: { status: AttendanceStatus }) {
    const { label, className } = statusConfig[status];
    return (
        <span
            className={cn(
                "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
                className
            )}
        >
            {label}
        </span>
    );
}
