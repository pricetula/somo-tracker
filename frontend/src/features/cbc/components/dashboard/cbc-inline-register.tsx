"use client";

import * as React from "react";
import { Loader2, Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type {
    AttendanceStatus,
    AttendanceStudentRow,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";

// ─── Status config ─────────────────────────────────────────────────────────

const STATUS_CONFIG: Record<AttendanceStatus, { label: string; activeClass: string }> = {
    PRESENT: {
        label: "P",
        activeClass: "bg-green-100 border-green-400 text-green-800",
    },
    ABSENT: {
        label: "A",
        activeClass: "bg-red-100 border-red-400 text-red-800",
    },
    LATE: {
        label: "L",
        activeClass: "bg-amber-100 border-amber-400 text-amber-800",
    },
    EXCUSED: {
        label: "E",
        activeClass: "bg-gray-100 border-gray-400 text-gray-600",
    },
};

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcInlineRegisterProps {
    students: AttendanceStudentRow[];
    periodId: string;
    localQueue: OfflineAttendanceEntry[];
    savingStudentIds: Set<string>;
    onSelectStatus: (studentId: string, status: AttendanceStatus, periodId: string) => void;
    onMarkAllPresent: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

function SaveIndicator({ state }: { state: "saving" | "saved" | "failed" }) {
    if (state === "saving") {
        return <Loader2 className="size-3 animate-spin text-amber-500" />;
    }
    if (state === "saved") {
        return <Check className="size-3 text-green-500" />;
    }
    return (
        <span className="inline-flex items-center gap-0.5 text-[10px] text-amber-600">
            <span className="inline-block size-1.5 rounded-full bg-amber-500" />
            not saved — retrying
        </span>
    );
}

export function CbcInlineRegister({
    students,
    periodId,
    localQueue,
    savingStudentIds,
    onSelectStatus,
    onMarkAllPresent,
}: CbcInlineRegisterProps) {
    const statuses: AttendanceStatus[] = ["PRESENT", "ABSENT", "LATE", "EXCUSED"];

    // ── Merge local queue ────────────────────────────────────────────
    const displayStudents = React.useMemo(() => {
        return students.map((student) => {
            const pendingEntry = localQueue.find((e) => e.studentId === student.student_id);
            if (pendingEntry) {
                return { ...student, status: pendingEntry.status, syncPending: true };
            }
            // Check if there's a failed entry in the queue (not in savingStudentIds)
            const isSaving = savingStudentIds.has(student.student_id);
            return { ...student, syncPending: false, _saveFailed: !isSaving };
        });
    }, [students, localQueue, savingStudentIds]);

    const unmarkedCount = displayStudents.filter((s) => s.status === null).length;

    return (
        <div className="border-t">
            {/* Bulk action bar */}
            {unmarkedCount > 0 && (
                <div className="flex items-center gap-2 border-b bg-gray-50 px-3 py-1.5">
                    <span className="text-muted-foreground text-[10px]">
                        {unmarkedCount} student{unmarkedCount !== 1 ? "s" : ""} not yet marked
                    </span>
                    <Button
                        variant="outline"
                        size="sm"
                        className="ml-auto h-6 px-2 text-[10px]"
                        onClick={onMarkAllPresent}
                    >
                        Mark all Present
                    </Button>
                </div>
            )}

            {/* Student list — compact for inline use */}
            <div className="max-h-64 overflow-y-auto">
                {displayStudents.map((student) => {
                    const isSaving = savingStudentIds.has(student.student_id);
                    const isFailed = localQueue.some(
                        (e) => e.studentId === student.student_id && !isSaving
                    );

                    return (
                        <div
                            key={student.student_id}
                            className={cn(
                                "flex items-center gap-2 border-b px-3 py-1.5 transition-colors last:border-b-0",
                                isFailed && "bg-amber-50"
                            )}
                        >
                            {/* Name */}
                            <div className="min-w-0 flex-1">
                                <p className="truncate text-xs font-medium">
                                    {student.student_name}
                                </p>
                            </div>

                            {/* Save indicator */}
                            {isSaving && <SaveIndicator state="saving" />}
                            {isFailed && <SaveIndicator state="failed" />}
                            {!isSaving && !isFailed && student.status !== null && (
                                <SaveIndicator state="saved" />
                            )}

                            {/* Status pills — compact (icon only) */}
                            <div className="flex shrink-0 gap-0.5">
                                {statuses.map((status) => {
                                    const cfg = STATUS_CONFIG[status];
                                    const isActive = student.status === status;

                                    return (
                                        <button
                                            key={status}
                                            type="button"
                                            onClick={() =>
                                                onSelectStatus(student.student_id, status, periodId)
                                            }
                                            disabled={isSaving}
                                            className={cn(
                                                "flex size-8 items-center justify-center rounded-md border text-xs font-bold transition-all",
                                                isActive
                                                    ? cfg.activeClass
                                                    : "text-muted-foreground border-transparent hover:bg-gray-50",
                                                isSaving && "cursor-not-allowed opacity-50"
                                            )}
                                            aria-label={`Mark ${student.student_name} as ${status.toLowerCase()}`}
                                            aria-pressed={isActive}
                                        >
                                            {cfg.label}
                                        </button>
                                    );
                                })}
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
