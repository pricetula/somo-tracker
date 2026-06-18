"use client";

import * as React from "react";

import { CbcAttendanceStudentRow } from "./cbc-attendance-student-row";
import type {
    AttendanceStudentRow,
    AttendanceStatus,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendanceGridProps {
    students: AttendanceStudentRow[];
    isLoading: boolean;
    periodId: string | null;
    /** Student statuses that have been updated optimistically. */
    localQueue: OfflineAttendanceEntry[];
    onSelectStatus: (studentId: string, status: AttendanceStatus, periodId: string) => void;
    savingStudentIds: Set<string>;
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendanceGrid({
    students,
    isLoading,
    periodId,
    localQueue,
    onSelectStatus,
    savingStudentIds,
}: CbcAttendanceGridProps) {
    // ── Merge local queue into the display ─────────────────────────────
    const displayStudents = React.useMemo(() => {
        return students.map((student) => {
            const pendingEntry = localQueue.find((e) => e.studentId === student.student_id);
            if (pendingEntry) {
                return {
                    ...student,
                    status: pendingEntry.status,
                    syncPending: true,
                };
            }
            return student;
        });
    }, [students, localQueue]);

    // ── Loading state ──────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="flex flex-col gap-1.5 px-3 py-4">
                {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className="bg-muted h-12 animate-pulse rounded-md" />
                ))}
            </div>
        );
    }

    // ── Empty state ────────────────────────────────────────────────────
    if (students.length === 0) {
        return (
            <div className="flex items-center justify-center py-8">
                <div className="text-center">
                    <p className="text-muted-foreground text-sm">
                        No students enrolled in this class for the current term.
                    </p>
                </div>
            </div>
        );
    }

    // ── No period selected (display students but disabled) ─────────────
    if (!periodId) {
        return (
            <div>
                <div className="border-border flex items-center border-b bg-gray-50 px-3 py-1.5">
                    <span className="text-muted-foreground text-xs font-medium">
                        {students.length} student{students.length !== 1 ? "s" : ""}
                    </span>
                    <span className="text-muted-foreground ml-auto text-xs italic">
                        Select a learning area to enable marking
                    </span>
                </div>
                {students.map((student) => (
                    <div
                        key={student.student_id}
                        className="flex items-center gap-3 border-b px-3 py-2.5 opacity-50"
                    >
                        <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-medium">{student.student_name}</p>
                        </div>
                    </div>
                ))}
            </div>
        );
    }

    // ── Active grid ────────────────────────────────────────────────────
    return (
        <div>
            {/* Header */}
            <div className="border-border flex items-center border-b bg-gray-50 px-3 py-1.5">
                <span className="text-muted-foreground text-xs font-medium">
                    {students.length} student{students.length !== 1 ? "s" : ""}
                </span>
            </div>

            {/* Rows */}
            {displayStudents.map((student) => (
                <CbcAttendanceStudentRow
                    key={student.student_id}
                    studentName={student.student_name}
                    admissionNumber={student.admission_number}
                    currentStatus={student.status}
                    isSaving={savingStudentIds.has(student.student_id)}
                    syncPending={student.syncPending ?? false}
                    onSelectStatus={(status) =>
                        onSelectStatus(student.student_id, status, periodId)
                    }
                />
            ))}
        </div>
    );
}
