"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";
import type { AttendanceStatus } from "@/features/cbc/types";

// ─── Status config ─────────────────────────────────────────────────────────

interface StatusConfig {
    label: string;
    icon: string;
    activeClass: string;
    inactiveClass: string;
}

const STATUS_CONFIG: Record<AttendanceStatus, StatusConfig> = {
    PRESENT: {
        label: "Present",
        icon: "✓",
        activeClass: "bg-green-100 border-green-400 text-green-800",
        inactiveClass: "hover:bg-green-50 border-transparent text-muted-foreground",
    },
    ABSENT: {
        label: "Absent",
        icon: "✕",
        activeClass: "bg-red-100 border-red-400 text-red-800",
        inactiveClass: "hover:bg-red-50 border-transparent text-muted-foreground",
    },
    LATE: {
        label: "Late",
        icon: "◷",
        activeClass: "bg-amber-100 border-amber-400 text-amber-800",
        inactiveClass: "hover:bg-amber-50 border-transparent text-muted-foreground",
    },
    EXCUSED: {
        label: "Excused",
        icon: "—",
        activeClass: "bg-gray-100 border-gray-400 text-gray-600",
        inactiveClass: "hover:bg-gray-50 border-transparent text-muted-foreground",
    },
};

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendanceStudentRowProps {
    studentName: string;
    admissionNumber?: string;
    currentStatus: AttendanceStatus | null;
    isSaving: boolean;
    syncPending: boolean;
    onSelectStatus: (status: AttendanceStatus) => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendanceStudentRow({
    studentName,
    admissionNumber,
    currentStatus,
    isSaving,
    syncPending,
    onSelectStatus,
}: CbcAttendanceStudentRowProps) {
    const statuses: AttendanceStatus[] = ["PRESENT", "ABSENT", "LATE", "EXCUSED"];

    return (
        <div
            className={cn(
                "flex items-center gap-3 border-b px-3 py-2.5 transition-colors",
                syncPending && "bg-amber-50"
            )}
        >
            {/* Student info */}
            <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{studentName}</p>
                {admissionNumber && (
                    <p className="text-muted-foreground truncate text-xs">{admissionNumber}</p>
                )}
            </div>

            {/* Sync pending badge */}
            {syncPending && (
                <div className="flex items-center gap-1 text-amber-600">
                    <Loader2 className="size-3 animate-spin" />
                    <span className="text-[10px]">saving...</span>
                </div>
            )}

            {/* Status toggle pills */}
            <div className="flex shrink-0 gap-1">
                {statuses.map((status) => {
                    const cfg = STATUS_CONFIG[status];
                    const isActive = currentStatus === status;
                    return (
                        <button
                            key={status}
                            type="button"
                            onClick={() => onSelectStatus(status)}
                            disabled={isSaving}
                            className={cn(
                                "flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-all",
                                "min-h-[40px] min-w-[40px]", // 40px minimum tap target
                                isActive ? cfg.activeClass : cfg.inactiveClass,
                                isSaving && "cursor-not-allowed opacity-50"
                            )}
                            aria-label={`Mark ${studentName} as ${cfg.label}`}
                            aria-pressed={isActive}
                            title={cfg.label}
                        >
                            <span className="text-sm">{cfg.icon}</span>
                            <span className="hidden sm:inline">{cfg.label}</span>
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
