"use client";

import * as React from "react";
import { AlertTriangle, Users } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { CbcAttendanceStudentRow } from "./cbc-attendance-student-row";
import type {
    AttendanceStudentRow,
    AttendanceStatus,
    CbcAttendanceLogDetail,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendanceGridProps {
    students: AttendanceStudentRow[];
    /** Full log details for recorder info. */
    logs: CbcAttendanceLogDetail[];
    isLoading: boolean;
    periodId: string | null;
    localQueue: OfflineAttendanceEntry[];
    onSelectStatus: (studentId: string, status: AttendanceStatus, periodId: string) => void;
    onRemarksChange: (studentId: string, remarks: string) => void;
    onMarkRemainingAsPresent: (studentIds: string[]) => void;
    savingStudentIds: Set<string>;
    /** Whether the current user can edit (SCHOOL_ADMIN always; TEACHER only for own periods). */
    canEdit: boolean;
    /** Recorded-by user ID — if different from current user, show read-only note. */
    recordedByUserId?: string;
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendanceGrid({
    students,
    logs,
    isLoading,
    periodId,
    localQueue,
    onSelectStatus,
    onRemarksChange,
    onMarkRemainingAsPresent,
    savingStudentIds,
    canEdit,
    recordedByUserId,
}: CbcAttendanceGridProps) {
    // ── Confirm-on-edit tracking (one-time per student per session) ───
    const [confirmedEdits, setConfirmedEdits] = React.useState<Set<string>>(new Set());
    const [pendingEdit, setPendingEdit] = React.useState<{
        studentId: string;
        status: AttendanceStatus;
    } | null>(null);

    // ── Merge local queue into display ────────────────────────────────
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

    // ── Build a map of student_id → log for recorder info ────────────
    const logMap = React.useMemo(() => {
        const map = new Map<string, CbcAttendanceLogDetail>();
        for (const log of logs) {
            map.set(log.student_id, log);
        }
        return map;
    }, [logs]);

    // ── Count unmarked students ──────────────────────────────────────
    const unmarkedStudentIds = React.useMemo(() => {
        return students.filter((s) => s.status === null).map((s) => s.student_id);
    }, [students]);

    // ── Handle status selection with confirm-on-edit ─────────────────
    const handleSelectStatus = (studentId: string, status: AttendanceStatus, pid: string) => {
        const student = students.find((s) => s.student_id === studentId);
        const hasExistingMark = student?.status !== null && student?.log_id !== null;
        const alreadyConfirmed = confirmedEdits.has(studentId);

        if (hasExistingMark && !alreadyConfirmed) {
            setPendingEdit({ studentId, status });
            return;
        }

        onSelectStatus(studentId, status, pid);
        if (hasExistingMark) {
            setConfirmedEdits((prev) => new Set(prev).add(studentId));
        }
    };

    const handleConfirmEdit = () => {
        if (!pendingEdit || !periodId) return;
        const { studentId, status } = pendingEdit;
        setConfirmedEdits((prev) => new Set(prev).add(studentId));
        setPendingEdit(null);
        onSelectStatus(studentId, status, periodId);
    };

    // ── Loading state ────────────────────────────────────────────────
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

    // ── No period selected ─────────────────────────────────────────────
    if (!periodId) {
        return (
            <div>
                <div className="border-border flex items-center border-b bg-gray-50 px-3 py-1.5">
                    <span className="text-muted-foreground text-xs font-medium">
                        <Users className="mr-1 inline size-3" />
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
    const isReadOnly = !canEdit;

    return (
        <div>
            {/* Header row */}
            <div className="border-border flex items-center border-b bg-gray-50 px-3 py-1.5">
                <span className="text-muted-foreground flex items-center gap-1 text-xs font-medium">
                    <Users className="size-3" />
                    {students.length} student{students.length !== 1 ? "s" : ""}
                    {unmarkedStudentIds.length > 0 && (
                        <span className="ml-1 text-amber-600">
                            · {unmarkedStudentIds.length} not marked
                        </span>
                    )}
                </span>

                {/* Bulk action: Mark remaining as Present */}
                {!isReadOnly && unmarkedStudentIds.length > 0 && (
                    <Button
                        variant="outline"
                        size="sm"
                        className="ml-auto h-7 px-2 text-[10px]"
                        onClick={() => onMarkRemainingAsPresent(unmarkedStudentIds)}
                    >
                        Mark remaining as Present
                    </Button>
                )}

                {isReadOnly && recordedByUserId && (
                    <span className="text-muted-foreground ml-auto text-[10px] italic">
                        View-only mode
                    </span>
                )}
            </div>

            {/* Student rows */}
            {displayStudents.map((student) => {
                const log = logMap.get(student.student_id);
                return (
                    <CbcAttendanceStudentRow
                        key={student.student_id}
                        studentName={student.student_name}
                        admissionNumber={student.admission_number}
                        currentStatus={student.status}
                        isSaving={savingStudentIds.has(student.student_id)}
                        syncPending={student.syncPending ?? false}
                        recordedByLabel={log?.recorded_by_label}
                        onSelectStatus={(status) =>
                            handleSelectStatus(student.student_id, status, periodId)
                        }
                        remarks={log?.remarks ?? null}
                        onRemarksChange={(remarks) => onRemarksChange(student.student_id, remarks)}
                        readOnly={isReadOnly}
                        readOnlyNote={
                            isReadOnly
                                ? "Recorded by another teacher — contact your admin to make changes"
                                : undefined
                        }
                    />
                );
            })}

            {/* ── Confirm-edit dialog (one-time per session per student) ── */}
            <Dialog
                open={pendingEdit !== null}
                onOpenChange={(open) => {
                    if (!open) setPendingEdit(null);
                }}
            >
                <DialogContent className="max-w-sm">
                    <DialogHeader>
                        <div className="flex items-center gap-2">
                            <AlertTriangle className="size-5 text-amber-500" />
                            <DialogTitle className="text-sm">Editing existing record</DialogTitle>
                        </div>
                        <DialogDescription className="text-xs">
                            You&apos;re about to change an already-submitted attendance mark. This
                            will update the attendance record for this student.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter className="gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setPendingEdit(null)}
                            className="text-xs"
                        >
                            Cancel
                        </Button>
                        <Button
                            variant="default"
                            size="sm"
                            onClick={handleConfirmEdit}
                            className="text-xs"
                        >
                            Continue editing
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
